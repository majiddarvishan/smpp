package smpp

//go:generate stringer -type=SessionState,SessionType

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/majiddarvishan/smpp/pdu"
)

var smppLogs bool

func init() {
	flag.BoolVar(&smppLogs, "smpp.logs", true, "show smpp logging")
}

// Error implements Error and Temporary interfaces.
type Error struct {
	Msg  string
	Temp bool
}

func (e Error) Error() string {
	return e.Msg
}

// Temporary implements Temporary interface.
func (e Error) Temporary() bool {
	return e.Temp
}

// SessionState describes session state.
type SessionState int

const (
	// StateOpen is the initial session state.
	StateOpen SessionState = iota
	// StateBinding session has started binding process.
	// All communication will be blocked until session is bound.
	StateBinding
	// StateBoundTx session is bound as transmitter.
	StateBoundTx
	// StateBoundRx session is bound as receiver.
	StateBoundRx
	// StateBoundTRx session is bound as transceiver.
	StateBoundTRx
	// StateUnbinding session has started unbinding process.
	// Prevents any communication until unbinding is finished.
	StateUnbinding
	// StateClosing session is gracefully closing.
	StateClosing
	// StateClosed session is closed.
	StateClosed
)

// SessionType defines if session is ESME or SMSC. In other words it defines
// if the session will behave like a client or like a server.
type SessionType int

const (
	// ESME type of the session.
	ESME SessionType = iota
	// SMSC type of the session.
	SMSC
)

// Logger provides logging interface for getting info about internals of smpp package.
type Logger interface {
	InfoF(msg string, params ...interface{})
	ErrorF(msg string, params ...interface{})
}

// DefaultLogger prints logs if smpp.logs flag is set.
type DefaultLogger struct{}

// InfoF implements Logger interface.
func (dl DefaultLogger) InfoF(msg string, params ...interface{}) {
	if smppLogs {
		log.Printf("INFO: "+msg+"\n", params...)
	}
}

// ErrorF implements Logger interface.
func (dl DefaultLogger) ErrorF(msg string, params ...interface{}) {
	if smppLogs {
		log.Printf("ERRO: "+msg+"\n", params...)
	}
}

// Handler handles smpp requests.
type Handler interface {
	ServeSMPP(ctx *Context)
}

// RequestHandlerFunc wraps func into Handler.
type RequestHandlerFunc func(ctx *Context)

// ServeSMPP implements Handler interface.
func (hc RequestHandlerFunc) ServeSMPP(ctx *Context) {
	hc(ctx)
}

// ResponseHandlerFunc wraps func into Handler.
type ResponseHandlerFunc func(ctx *Context)

// ServeSMPP implements Handler interface.
func (hc ResponseHandlerFunc) ServeSMPP(ctx *Context) {
	hc(ctx)
}

type defaultHandler struct{}

func (h defaultHandler) ServeSMPP(ctx *Context) {
	ctx.Respond(&pdu.GenericNack{}, pdu.StatusSysErr)
}

func genSessionID() string {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%X-%X-%X", b[0:4], b[4:6], b[6:8])
}

// RemoteAddresser is an abstraction to keep Session from depending
// on network connection.
type RemoteAddresser interface {
	RemoteAddr() net.Addr
}

// SessionConf structured session configuration.
type SessionConf struct {
	Type            SessionType
	SendWinSize     int
	ReqWinSize      int
	WindowTimeout   time.Duration
	SessionState    func(sessionID, systemID string, state SessionState)
	SystemID        string
	ID              string
	Logger          Logger
	RequestHandler  Handler
	ResponseHandler Handler
	Sequencer       pdu.Sequencer
	// MapResetInterval specifies the duration after which the session's map will be recreated
	// to mitigate potential memory growth. Setting this to a positive duration can help
	// manage memory usage, especially when large amounts of data are added and removed from the map.
	MapResetInterval time.Duration
}

type response struct {
	hdr  pdu.Header
	resp pdu.PDU
	err  error
}

// Session is the engine that coordinates SMPP protocol for bounded peers.
type Session struct {
	conf     *SessionConf
	RWC      io.ReadWriteCloser
	enc      *pdu.Encoder
	dec      *pdu.Decoder
	wg       sync.WaitGroup
	mu       sync.Mutex
	seq      uint32
	reqCount int
	sent     map[uint32]chan response
	state    SessionState
	systemID string
	closed   chan struct{}
}

// NewSession creates new SMPP session and starts goroutine for listening incoming
// requests so make sure to call Session.Close() after you are done using it to
// avoid goroutine leak.
// Session will take ownership of the ReadWriteCloser and call Close on it during
// shutdown.
func NewSession(rwc io.ReadWriteCloser, conf SessionConf) *Session {
	if conf.SendWinSize == 0 {
		conf.SendWinSize = 10
	}
	if conf.Logger == nil {
		conf.Logger = DefaultLogger{}
	}
	if conf.RequestHandler == nil {
		conf.RequestHandler = &defaultHandler{}
	}
	if conf.ResponseHandler == nil {
		conf.ResponseHandler = &defaultHandler{}
	}
	if conf.WindowTimeout == 0 {
		conf.WindowTimeout = 10 * time.Second
	}
	if conf.ReqWinSize == 0 {
		conf.ReqWinSize = 10
	}
	if conf.ID == "" {
		conf.ID = genSessionID()
	}

	if conf.MapResetInterval == 0 {
		conf.MapResetInterval = time.Hour * 12
	}
	sess := &Session{
		conf:   &conf,
		RWC:    rwc,
		enc:    pdu.NewEncoder(conf.Sequencer),
		dec:    pdu.NewDecoder(),
		sent:   make(map[uint32]chan response, conf.SendWinSize),
		closed: make(chan struct{}),
	}
	sess.wg.Add(1)
	go sess.serve()
	go sess.resetSentMapPeriodically()
	return sess
}

// ID uniquely identifies the session.
func (sess *Session) ID() string {
	return sess.conf.ID
}

// SystemID identifies connected peer.
func (sess *Session) SystemID() string {
	if sess.conf.SystemID != "" {
		return sess.conf.SystemID
	}
	if sess.systemID != "" {
		return sess.systemID
	}
	return "-"
}

func (sess *Session) String() string {
	return fmt.Sprintf("(%s:%s:%s)", sess.conf.Type, sess.SystemID(), sess.conf.ID)
}

func (sess *Session) remoteAddr() string {
	if ra, ok := sess.RWC.(RemoteAddresser); ok {
		return ra.RemoteAddr().String()
	}
	return ""
}

// serve handles incoming PDU by decoding it and delegating processing to the handler
// if it's the request or handling it over to the sender if it's a response.
func (sess *Session) serve() {
	defer sess.wg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// todo: check what are these and why they are deleted
	// sess.wg.Add(1)
	// go sess.resetSentMapPeriodically(ctx)

	for {
		// Read header first.
		var headerBytes [16]byte
		if _, err := io.ReadFull(sess.RWC, headerBytes[:]); err != nil {
			if err == io.EOF {
				sess.conf.Logger.InfoF("decoding pdu: %s %+v", sess, err)
			} else {
				sess.conf.Logger.ErrorF("decoding pdu: %s %+v", sess, err)
			}
			sess.shutdown()
			return
		}
		h, p, err := sess.dec.DecodeHeader(headerBytes[:])
		if err != nil {
			if err == io.EOF {
				sess.conf.Logger.InfoF("decoding pdu: %s %+v", sess, err)
			} else {
				sess.conf.Logger.ErrorF("decoding pdu: %s %+v", sess, err)
			}
			sess.shutdown()
			return
		}
		if h.Length() > 16 {
			bodyBytes := make([]byte, h.Length()-16)
			if len(bodyBytes) > 0 {
				if _, err := io.ReadFull(sess.RWC, bodyBytes); err != nil {
					// sess.conf.Logger.Errorf("smpp: pdu length doesn't match read body length %d != %d", h.Length(), len(bodyBytes))
                    sess.conf.Logger.ErrorF("smpp: pdu length doesn't match read body length %d != %d", h.Length(), len(bodyBytes))
					sess.shutdown()
					return
				}
			}
			// Unmarshal binary
			if err := p.UnmarshalBinary(bodyBytes); err != nil {
				if err == io.EOF {
					sess.conf.Logger.InfoF("decoding pdu: %s %+v", sess, err)
				} else {
					sess.conf.Logger.ErrorF("decoding pdu: %s %+v", sess, err)
				}
				sess.shutdown()
				return
			}
		}

		sess.mu.Lock()

		// todo: I have to do better implementation
		switch h.CommandID() {
		case pdu.BindTransceiverID, pdu.BindTransmitterID, pdu.BindReceiverID:
			sess.systemID = pdu.SystemID(p)
		}

		if err := sess.makeTransition(h.CommandID(), true); err != nil {
			sess.conf.Logger.ErrorF("transitioning upon receive: %s %+v", sess, err)
			sess.mu.Unlock()
			continue
		}
		// Handle PDU requests.
		if pdu.IsRequest(h.CommandID()) {
			// sess.conf.Logger.InfoF("received request: %s %s%+v, header: %#v", sess, p.CommandID(), p, h)
			sess.conf.Logger.InfoF("received request: %s %s, \nheader:\n%vbody\n%+v", sess, p.CommandID(), h, p)
			if sess.reqCount == sess.conf.ReqWinSize {
				sess.throttle(h.Sequence())
			} else {
				sess.wg.Add(1)
				sess.reqCount++
				go sess.handleRequest(ctx, h, p)
			}
			sess.mu.Unlock()
			continue
		}
		// Handle PDU responses.
		// if l, ok := sess.sent[h.Sequence()]; ok {
		if _, ok := sess.sent[h.Sequence()]; ok {
			// sess.conf.Logger.InfoF("received response: %s %s%+v, header: %#v", sess, p.CommandID(), p, h)
			sess.conf.Logger.InfoF("received response: %s %s, \nheader:\n%vbody\n%+v", sess, p.CommandID(), h, p)
			delete(sess.sent, h.Sequence())

			sess.wg.Add(1)
			go sess.handleResponse(ctx, h, p)

			sess.mu.Unlock()

			// l <- response{
			// 	hdr:  h,
			// 	resp: p,
			// 	err:  toError(h.Status()),
			// }
			continue
		}
		sess.conf.Logger.ErrorF("unexpected response: %s %s%+v, header: %#v", sess, p.CommandID(), p, h)
		sess.mu.Unlock()
	}
}

func (sess *Session) throttle(seq uint32) {
	resp := pdu.GenericNack{}
	_, buf, err := sess.enc.Encode(resp, pdu.EncodeStatus(pdu.StatusThrottled), pdu.EncodeSeq(seq))
	if err != nil {
		sess.conf.Logger.ErrorF("error encoding pdu: %s %+v", sess, err)
		return
	}

	_, err = sess.RWC.Write(buf)
	if err != nil {
		sess.conf.Logger.ErrorF("error sending GenericNack: %s %+v", sess, err)
	}
}

func (sess *Session) handleRequest(ctx context.Context, h pdu.Header, req pdu.PDU) {
	ctx, cancel := context.WithTimeout(ctx, sess.conf.WindowTimeout)
	defer func() {
		cancel()
		sess.mu.Lock()
		sess.reqCount--
		sess.mu.Unlock()
		sess.wg.Done()
	}()
	sessCtx := &Context{
		Sess: sess,
		ctx:  ctx,
		seq:  h.Sequence(),
		hdr:  h,
		pdu:  req,
	}
	sess.conf.RequestHandler.ServeSMPP(sessCtx)

	if sessCtx.close {
		sess.shutdown()
	}
}

func (sess *Session) handleResponse(ctx context.Context, h pdu.Header, resp pdu.PDU) {
    defer func() {
		// sess.mu.Lock()
		// sess.reqCount--
		// sess.mu.Unlock()
		sess.wg.Done()
	}()

	sessCtx := &Context{
		Sess: sess,
		ctx:  ctx,
		seq:  h.Sequence(),
		hdr:  h,
		pdu:  resp,
	}

	sess.conf.ResponseHandler.ServeSMPP(sessCtx)

	if sessCtx.close {
		sess.shutdown()
	}
}

func (sess *Session) shutdown() {
	go sess.Close()
}

// Close implements Closer interface. It MUST be called to dispose session cleanly.
// It gracefully waits for all handlers to finish execution before returning.
func (sess *Session) Close() error {
	sess.mu.Lock()
	if err := sess.setState(StateClosing); err != nil {
		sess.mu.Unlock()
		return err
	}
	for k, l := range sess.sent {
		delete(sess.sent, k)
		close(l)
	}
	sess.RWC.Close()
	if err := sess.setState(StateClosed); err != nil {
		sess.mu.Unlock()
		return err
	}
	sess.mu.Unlock()
	sess.wg.Wait()
	sess.conf.Logger.InfoF("session closed: %s", sess)
	close(sess.closed)
	return nil
}

// Must be guarded by mutex.
func (sess *Session) setState(state SessionState) error {
    sess.conf.Logger.InfoF("smpp: old state (%s) wants change to new state (%s)", sess.state.String(), state.String())

	if sess.state == state {
		return fmt.Errorf("smpp: setting same state twice %s", state)
	}
	switch sess.state {
	case StateOpen:
		// if state != StateBinding && state != StateClosing {
		if state != StateBinding {
			return fmt.Errorf("smpp: setting open session to invalid state %s", state)
		}
	case StateBinding:
		switch state {
		case StateOpen, StateBoundRx, StateBoundTRx, StateBoundTx:
        case StateClosing:
		default:
			return fmt.Errorf("smpp: setting binding session to invalid state %s", state)
		}
	case StateBoundRx, StateBoundTRx, StateBoundTx:
		switch state {
		case StateUnbinding, StateClosing:
		default:
			return fmt.Errorf("smpp: setting bound session to invalid state %s", state)
		}
	case StateUnbinding:
		if state != StateClosing {
			return fmt.Errorf("smpp: setting unbinding session to invalid state %s", state)
		}
	case StateClosing:
		if state != StateClosed {
			return fmt.Errorf("smpp: setting closing session to invalid state %s", state)
		}
	case StateClosed:
		return fmt.Errorf("smpp: session %s already in closed state %s", sess, state)
	}
	sess.state = state
	if hook := sess.conf.SessionState; hook != nil {
		hook(sess.conf.ID, sess.SystemID(), sess.state)
	}
	return nil
}

// Send writes PDU to the bounded connection effectively sending it to the peer.
// Use context deadline to specify how much you would like to wait for the response.
func (sess *Session) SendRequest(ctx context.Context, req pdu.PDU, opts ...pdu.EncoderOption) (uint32, error) {
	if req == nil {
		return 0, Error{Msg: "smpp: sending nil pdu"}
	}
	sess.mu.Lock()
	if len(sess.sent) == sess.conf.SendWinSize {
		sess.mu.Unlock()
		return 0, Error{Msg: "smpp: sending window closed", Temp: true}
	}
	if err := sess.makeTransition(req.CommandID(), false); err != nil {
		sess.conf.Logger.ErrorF("transitioning before send: %s %+v", sess, err)
		sess.mu.Unlock()
		return 0, err
	}
	seq, buf, err := sess.enc.Encode(req, opts...)
	if err != nil {
		sess.mu.Unlock()
		return 0, err
	}
	l := make(chan response, 1)
	sess.sent[seq] = l

	_, err = sess.RWC.Write(buf)
	if err != nil {
		delete(sess.sent, seq)

		sess.mu.Unlock()
		return 0, err
	}

	// sess.conf.Logger.InfoF("request sent: %s %s%+v", sess, req.CommandID(), req)
	// sess.conf.Logger.InfoF("request sent: %s %s, \nheader:\n%vbody\n%+v", sess, req.CommandID(), ctx.hdr, req)
	sess.conf.Logger.InfoF("request sent: %s %s, \nsequence: %d\nbody\n%+v", sess, req.CommandID(), seq, req)
	sess.mu.Unlock()
	return seq, nil
	// select {
	// case resp, ok := <-l:
	// 	if !ok {
	// 		return nil, nil, SessionClosedBeforeReceiving
	// 	}
	// 	if resp.err != nil {
	// 		return resp.hdr, resp.resp, resp.err
	// 	}
	// 	return resp.hdr, resp.resp, nil
	// case <-ctx.Done():
	// 	return nil, nil, ctx.Err()
	// }
}

func (sess *Session) SendResponse(ctx *Context, resp pdu.PDU, status pdu.Status) error {
	sess.mu.Lock()
	if err := sess.makeTransition(resp.CommandID(), false); err != nil {
		sess.conf.Logger.ErrorF("transitioning resp pdu: %s %+v", sess, err)
		sess.mu.Unlock()
		return err
	}

	_, buf, err := sess.enc.Encode(resp, pdu.EncodeStatus(status), pdu.EncodeSeq(ctx.seq))
	if err != nil {
		sess.conf.Logger.ErrorF("error encoding pdu: %s %+v", sess, err)
		sess.mu.Unlock()
		return err
	}

	_, err = sess.RWC.Write(buf)
	if err != nil {
		sess.mu.Unlock()
		return err
	}

	// ctx.Sess.conf.Logger.InfoF("sent response: %s %s %+v", ctx.Sess, resp.CommandID(), resp)
	sess.conf.Logger.InfoF("sent response: %s %s, \nheader:\n%vbody\n%+v", sess, resp.CommandID(), ctx, resp)
	sess.mu.Unlock()
	return nil
}

// makeTransition checks if processing pdu ID in the current session state is valid operation,
// if yes it transitions state to the new one triggered by ID.
//
// Must be guarded by mutex.
func (sess *Session) makeTransition(ID pdu.CommandID, received bool) error {
	// If sending from ESME or receiving on SMSC we have the same rules.
	if (sess.conf.Type == ESME && !received) || (sess.conf.Type == SMSC && received) {
		switch sess.state {
		case StateOpen:
			switch ID {
			case pdu.BindTransceiverID, pdu.BindTransmitterID, pdu.BindReceiverID:
				return sess.setState(StateBinding)
			}
		case StateBinding:
			if ID == pdu.GenericNackID {
				return sess.setState(StateOpen)
			}
		case StateBoundTx:
			switch ID {
			case pdu.UnbindID:
				return sess.setState(StateUnbinding)
			case pdu.UnbindRespID, pdu.DeliverSmRespID, pdu.DataSmID, pdu.SubmitSmID, pdu.SubmitMultiID,
				pdu.DataSmRespID, pdu.EnquireLinkID, pdu.EnquireLinkRespID, pdu.ReplaceSmID,
				pdu.GenericNackID:
				return nil
			}
		case StateBoundRx:
			switch ID {
			case pdu.UnbindID:
				return sess.setState(StateUnbinding)
			case pdu.UnbindRespID, pdu.DeliverSmRespID, pdu.DataSmID,
				pdu.DataSmRespID, pdu.EnquireLinkID, pdu.EnquireLinkRespID,
				pdu.GenericNackID:
				return nil
			}
		case StateBoundTRx:
			switch ID {
			case pdu.UnbindID:
				return sess.setState(StateUnbinding)
			case pdu.SubmitSmID, pdu.SubmitSmRespID, pdu.DeliverSmRespID,
				pdu.DataSmID, pdu.DataSmRespID, pdu.EnquireLinkID, pdu.EnquireLinkRespID, pdu.SubmitMultiID, pdu.SubmitMultiRespID,
				pdu.QuerySmID, pdu.CancelSmID, pdu.GenericNackID:
				return nil
			}
		case StateUnbinding:
			if ID == pdu.UnbindRespID {
				return nil
			}
		case StateClosing, StateClosed:
		}
		// If sending from SMSC or receiving on ESME we have the same rules.
	} else if (sess.conf.Type == SMSC && !received) || (sess.conf.Type == ESME && received) {
		switch sess.state {
		case StateOpen:
			switch ID {
			case pdu.OutbindID:
				return nil
			}
		case StateBinding:
			switch ID {
			case pdu.BindTransceiverRespID:
				return sess.setState(StateBoundTRx)
			case pdu.BindTransmitterRespID:
				return sess.setState(StateBoundTx)
			case pdu.BindReceiverRespID:
				return sess.setState(StateBoundRx)
			case pdu.GenericNackID:
				return sess.setState(StateOpen)
			}
		case StateBoundTx:
			switch ID {
			case pdu.UnbindID:
				return sess.setState(StateUnbinding)
			case pdu.SubmitSmRespID, pdu.SubmitMultiRespID, pdu.DataSmID, pdu.DataSmRespID,
				pdu.QuerySmRespID, pdu.CancelSmRespID, pdu.ReplaceSmRespID, pdu.EnquireLinkID, pdu.EnquireLinkRespID,
				pdu.GenericNackID:
				return nil
			}
		case StateBoundRx:
			switch ID {
			case pdu.UnbindID:
				return sess.setState(StateUnbinding)
			case pdu.DeliverSmID, pdu.DataSmID, pdu.DataSmRespID,
				pdu.EnquireLinkID, pdu.EnquireLinkRespID, pdu.AlertNotificationID, pdu.GenericNackID:
				return nil
			}
		case StateBoundTRx:
			switch ID {
			case pdu.UnbindID:
				return sess.setState(StateUnbinding)
			case pdu.SubmitSmRespID, pdu.SubmitMultiRespID, pdu.DataSmID, pdu.DataSmRespID, pdu.DeliverSmID,
				pdu.QuerySmRespID, pdu.CancelSmRespID, pdu.AlertNotificationID, pdu.ReplaceSmRespID, pdu.EnquireLinkID, pdu.EnquireLinkRespID,
				pdu.GenericNackID:
				return nil
			}
		case StateUnbinding:
			if ID == pdu.UnbindRespID {
				return nil
			}
		case StateClosing, StateClosed:
		}
	}
	return Error{Msg: fmt.Sprintf("smpp: processing '%s' in invalid session state '%s'", ID, sess.state), Temp: true}
}

// NotifyClosed provides channel that will be closed once session enters closed state.
func (sess *Session) NotifyClosed() <-chan struct{} {
	return sess.closed
}

func (sess *Session) resetSentMapPeriodically() {
	ticker := time.NewTicker(sess.conf.MapResetInterval)
	defer ticker.Stop()
	for {
		select {
		case <-sess.closed:
			return
		case <-ticker.C:
			sess.mu.Lock()
			// Creating a new map to address the Go map's memory behavior where
			// the underlying memory does not shrink even after entries are deleted.
			// This helps in releasing memory occupied by old map's buckets.
			// For a deeper dive into the memory behavior of Go maps, you can refer to:
			// https://teivah.medium.com/maps-and-memory-leaks-in-go-a85ebe6e7e69
			newSent := make(map[uint32]chan response)
			for i, responses := range sess.sent {
				newSent[i] = responses
			}
			sess.sent = newSent
			sess.mu.Unlock()
		}
	}
}

// StatusError implements error interface for SMPP status errors.
type StatusError struct {
	msg    string
	status pdu.Status
}

// Error implements error interface.
func (se StatusError) Error() string {
	return fmt.Sprintf("%s '0x%X'", se.msg, int(se.status))
}

// Status returns PDU status code of the error.
func (se StatusError) Status() pdu.Status {
	return se.status
}

func toError(status pdu.Status) error {
	switch status {
	case pdu.StatusOK:
		return nil
	case pdu.StatusInvMsgLen:
		return StatusError{"Message Length is invalid", pdu.StatusInvMsgLen}
	case pdu.StatusInvCmdLen:
		return StatusError{"Command Length is invalid", pdu.StatusInvCmdLen}
	case pdu.StatusInvCmdID:
		return StatusError{"Invalid Command ID", pdu.StatusInvCmdID}
	case pdu.StatusInvBnd:
		return StatusError{"Incorrect BIND Status for given command", pdu.StatusInvBnd}
	case pdu.StatusAlyBnd:
		return StatusError{"ESME Already in Bound State", pdu.StatusAlyBnd}
	case pdu.StatusInvPrtFlg:
		return StatusError{"Invalid Priority Flag", pdu.StatusInvPrtFlg}
	case pdu.StatusInvRegDlvFlg:
		return StatusError{"Invalid Registered Delivery Flag", pdu.StatusInvRegDlvFlg}
	case pdu.StatusSysErr:
		return StatusError{"System Error", pdu.StatusSysErr}
	case pdu.StatusInvSrcAdr:
		return StatusError{"Invalid Source Address", pdu.StatusInvSrcAdr}
	case pdu.StatusInvDstAdr:
		return StatusError{"Invalid Destination Address", pdu.StatusInvDstAdr}
	case pdu.StatusInvMsgID:
		return StatusError{"Message ID is invalid", pdu.StatusInvMsgID}
	case pdu.StatusBindFail:
		return StatusError{"Bind Failed", pdu.StatusBindFail}
	case pdu.StatusInvPaswd:
		return StatusError{"Invalid Password", pdu.StatusInvPaswd}
	case pdu.StatusInvSysID:
		return StatusError{"Invalid System ID", pdu.StatusInvSysID}
	case pdu.StatusCancelFail:
		return StatusError{"Cancel SM Failed", pdu.StatusCancelFail}
	case pdu.StatusReplaceFail:
		return StatusError{"Replace SM Failed", pdu.StatusReplaceFail}
	case pdu.StatusMsgQFul:
		return StatusError{"Message Queue Full", pdu.StatusMsgQFul}
	case pdu.StatusInvSerTyp:
		return StatusError{"Invalid Service Type", pdu.StatusInvSerTyp}
	case pdu.StatusInvNumDe:
		return StatusError{"Invalid number of destinations", pdu.StatusInvNumDe}
	case pdu.StatusInvDLName:
		return StatusError{"Invalid Distribution List name", pdu.StatusInvDLName}
	case pdu.StatusInvDestFlag:
		return StatusError{"Destination flag (submit_multi)", pdu.StatusInvDestFlag}
	case pdu.StatusInvSubRep:
		return StatusError{"Invalid ‘submit with replace’ request", pdu.StatusInvSubRep}
	case pdu.StatusInvEsmClass:
		return StatusError{"Invalid esm_class field data", pdu.StatusInvEsmClass}
	case pdu.StatusCntSubDL:
		return StatusError{"Cannot Submit to Distribution List", pdu.StatusCntSubDL}
	case pdu.StatusSubmitFail:
		return StatusError{"submit_sm or submit_multi failed", pdu.StatusSubmitFail}
	case pdu.StatusInvSrcTON:
		return StatusError{"Invalid Source address TON", pdu.StatusInvSrcTON}
	case pdu.StatusInvSrcNPI:
		return StatusError{"Invalid Source address NPI", pdu.StatusInvSrcNPI}
	case pdu.StatusInvDstTON:
		return StatusError{"Invalid Destination address TON", pdu.StatusInvDstTON}
	case pdu.StatusInvDstNPI:
		return StatusError{"Invalid Destination address NPI", pdu.StatusInvDstNPI}
	case pdu.StatusInvSysTyp:
		return StatusError{"Invalid system_type field", pdu.StatusInvSysTyp}
	case pdu.StatusInvRepFlag:
		return StatusError{"Invalid replace_if_present flag", pdu.StatusInvRepFlag}
	case pdu.StatusInvNumMsgs:
		return StatusError{"Invalid number of messages", pdu.StatusInvNumMsgs}
	case pdu.StatusThrottled:
		return StatusError{"Throttling error (ESME has exceeded allowed message limits)", pdu.StatusThrottled}
	case pdu.StatusInvSched:
		return StatusError{"Invalid Scheduled Delivery Time", pdu.StatusInvSched}
	case pdu.StatusInvExpiry:
		return StatusError{"Invalid message Expiry time", pdu.StatusInvExpiry}
	case pdu.StatusInvDftMsgID:
		return StatusError{"Predefined Message Invalid or Not Found", pdu.StatusInvDftMsgID}
	case pdu.StatusTempAppErr:
		return StatusError{"ESME Receiver Temporary App Error Code", pdu.StatusTempAppErr}
	case pdu.StatusPermAppErr:
		return StatusError{"ESME Receiver Permanent App Error Code", pdu.StatusPermAppErr}
	case pdu.StatusRejeAppErr:
		return StatusError{"ESME Receiver Reject Message Error Code", pdu.StatusRejeAppErr}
	case pdu.StatusQueryFail:
		return StatusError{"query_sm request failed", pdu.StatusQueryFail}
	case pdu.StatusInvOptParStream:
		return StatusError{"Error in the optional part of the PDU Body.", pdu.StatusInvOptParStream}
	case pdu.StatusOptParNotAllwd:
		return StatusError{"Optional Parameter not allowed", pdu.StatusOptParNotAllwd}
	case pdu.StatusInvParLen:
		return StatusError{"Invalid Parameter Length.", pdu.StatusInvParLen}
	case pdu.StatusMissingOptParam:
		return StatusError{"Expected Optional Parameter missing", pdu.StatusMissingOptParam}
	case pdu.StatusInvOptParamVal:
		return StatusError{"Invalid Optional Parameter Value", pdu.StatusInvOptParamVal}
	case pdu.StatusDeliveryFailure:
		return StatusError{"Delivery Failure", pdu.StatusDeliveryFailure}
	case pdu.StatusUnknownErr:
		return StatusError{"Unknown Error", pdu.StatusUnknownErr}
	}
	return StatusError{"Unknown Status", status}
}
