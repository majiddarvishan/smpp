package smpp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/majiddarvishan/smpp/pdu"
)

// Context represents container for SMPP request related information.
type Context struct {
	Sess   *Session
	status pdu.Status
	ctx    context.Context
	seq    uint32
	hdr    pdu.Header
	pdu    pdu.PDU
	close  bool
}

//todo: it's not correct place
func (h *Context) String() string {
    var sb strings.Builder
    sb.WriteString("{\n")
    // sb.WriteString(fmt.Sprintf(" length: %d\n", h.length))
    // sb.WriteString(fmt.Sprintf(" commandID: %d\n", h.commandID))
    sb.WriteString(fmt.Sprintf(" status: %d\n", h.status))
    sb.WriteString(fmt.Sprintf(" sequence: %d\n", h.seq))
    sb.WriteString("}\n")
    return sb.String()
}

func (ctx *Context) DebugReq() string {
	return fmt.Sprintf("%#v", ctx.pdu)
}

// SystemID returns SystemID of the bounded peer that request came from.
func (ctx *Context) SystemID() string {
	return ctx.Sess.conf.SystemID
}

// SessionID returns ID of the session that this context is responsible for handling this request.
func (ctx *Context) SessionID() string {
	return ctx.Sess.ID()
}

// CommandID returns ID of the PDU request.
func (ctx *Context) CommandID() pdu.CommandID {
	return ctx.pdu.CommandID()
}

// RemoteAddr returns IP address of the bounded peer.
func (ctx *Context) RemoteAddr() string {
	return ctx.Sess.remoteAddr()
}

// Context returns Go standard library context.
func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

// Status returns status of the current request.
func (ctx *Context) Status() pdu.Status {
	return ctx.status
}

// Status returns status of the current request.
func (ctx *Context) Sequence() uint32 {
	return ctx.seq
}

// CloseSession will initiate session shutdown after handler returns.
func (ctx *Context) CloseSession() {
	ctx.close = true
}

func (ctx *Context) Header() pdu.Header {
	return ctx.hdr
}

// GenericNack returns generic request PDU as pdu.GenericNack.
func (ctx *Context) GenericNack() (*pdu.GenericNack, error) {
	if p, ok := ctx.pdu.(*pdu.GenericNack); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// BindRx returns generic request PDU as pdu.BindRx.
func (ctx *Context) BindRx() (*pdu.BindRx, error) {
	if p, ok := ctx.pdu.(*pdu.BindRx); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// BindRxResp returns generic request PDU as pdu.BindRxResp.
func (ctx *Context) BindRxResp() (*pdu.BindRxResp, error) {
	if p, ok := ctx.pdu.(*pdu.BindRxResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// BindTx returns generic request PDU as pdu.BindTx.
func (ctx *Context) BindTx() (*pdu.BindTx, error) {
	if p, ok := ctx.pdu.(*pdu.BindTx); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// BindTxResp returns generic request PDU as pdu.BindTxResp.
func (ctx *Context) BindTxResp() (*pdu.BindTxResp, error) {
	if p, ok := ctx.pdu.(*pdu.BindTxResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// QuerySm returns generic request PDU as pdu.QuerySm.
func (ctx *Context) QuerySm() (*pdu.QuerySm, error) {
	if p, ok := ctx.pdu.(*pdu.QuerySm); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// QuerySmResp returns generic request PDU as pdu.QuerySmResp.
func (ctx *Context) QuerySmResp() (*pdu.QuerySmResp, error) {
	if p, ok := ctx.pdu.(*pdu.QuerySmResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// SubmitSm returns generic request PDU as pdu.SubmitSm.
func (ctx *Context) SubmitSm() (*pdu.SubmitSm, error) {
	if p, ok := ctx.pdu.(*pdu.SubmitSm); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// SubmitSmResp returns generic request PDU as pdu.SubmitSmResp.
func (ctx *Context) SubmitSmResp() (*pdu.SubmitSmResp, error) {
	if p, ok := ctx.pdu.(*pdu.SubmitSmResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// DeliverSm returns generic request PDU as pdu.DeliverSm.
func (ctx *Context) DeliverSm() (*pdu.DeliverSm, error) {
	if p, ok := ctx.pdu.(*pdu.DeliverSm); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// DeliverSmResp returns generic request PDU as pdu.DeliverSmResp.
func (ctx *Context) DeliverSmResp() (*pdu.DeliverSmResp, error) {
	if p, ok := ctx.pdu.(*pdu.DeliverSmResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// Unbind returns generic request PDU as pdu.Unbind.
func (ctx *Context) Unbind() (*pdu.Unbind, error) {
	if p, ok := ctx.pdu.(*pdu.Unbind); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// UnbindResp returns generic request PDU as pdu.UnbindResp.
func (ctx *Context) UnbindResp() (*pdu.UnbindResp, error) {
	if p, ok := ctx.pdu.(*pdu.UnbindResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// ReplaceSm returns generic request PDU as pdu.ReplaceSm.
func (ctx *Context) ReplaceSm() (*pdu.ReplaceSm, error) {
	if p, ok := ctx.pdu.(*pdu.ReplaceSm); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// ReplaceSmResp returns generic request PDU as pdu.ReplaceSmResp.
func (ctx *Context) ReplaceSmResp() (*pdu.ReplaceSmResp, error) {
	if p, ok := ctx.pdu.(*pdu.ReplaceSmResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// CancelSm returns generic request PDU as pdu.CancelSm.
func (ctx *Context) CancelSm() (*pdu.CancelSm, error) {
	if p, ok := ctx.pdu.(*pdu.CancelSm); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// CancelSmResp returns generic request PDU as pdu.CancelSmResp.
func (ctx *Context) CancelSmResp() (*pdu.CancelSmResp, error) {
	if p, ok := ctx.pdu.(*pdu.CancelSmResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// BindTRx returns generic request PDU as pdu.BindTRx.
func (ctx *Context) BindTRx() (*pdu.BindTRx, error) {
	if p, ok := ctx.pdu.(*pdu.BindTRx); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// BindTRxResp returns generic request PDU as pdu.BindTRxResp.
func (ctx *Context) BindTRxResp() (*pdu.BindTRxResp, error) {
	if p, ok := ctx.pdu.(*pdu.BindTRxResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// Outbind returns generic request PDU as pdu.Outbind.
func (ctx *Context) Outbind() (*pdu.Outbind, error) {
	if p, ok := ctx.pdu.(*pdu.Outbind); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// EnquireLink returns generic request PDU as pdu.EnquireLink.
func (ctx *Context) EnquireLink() (*pdu.EnquireLink, error) {
	if p, ok := ctx.pdu.(*pdu.EnquireLink); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// EnquireLinkResp returns generic request PDU as pdu.EnquireLinkResp.
func (ctx *Context) EnquireLinkResp() (*pdu.EnquireLinkResp, error) {
	if p, ok := ctx.pdu.(*pdu.EnquireLinkResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// SubmitMulti returns generic request PDU as pdu.SubmitMulti.
func (ctx *Context) SubmitMulti() (*pdu.SubmitMulti, error) {
	if p, ok := ctx.pdu.(*pdu.SubmitMulti); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// SubmitMultiResp returns generic request PDU as pdu.SubmitMultiResp.
func (ctx *Context) SubmitMultiResp() (*pdu.SubmitMultiResp, error) {
	if p, ok := ctx.pdu.(*pdu.SubmitMultiResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// AlertNotification returns generic request PDU as pdu.AlertNotification.
func (ctx *Context) AlertNotification() (*pdu.AlertNotification, error) {
	if p, ok := ctx.pdu.(*pdu.AlertNotification); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// DataSm returns generic request PDU as pdu.DataSm.
func (ctx *Context) DataSm() (*pdu.DataSm, error) {
	if p, ok := ctx.pdu.(*pdu.DataSm); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// DataSmResp returns generic request PDU as pdu.DataSmResp.
func (ctx *Context) DataSmResp() (*pdu.DataSmResp, error) {
	if p, ok := ctx.pdu.(*pdu.DataSmResp); ok {
		return p, nil
	}
	return nil, fmt.Errorf("smpp: invalid cast PDU is of type %s", ctx.pdu.CommandID())
}

// Respond sends pdu to the bounded peer.
func (ctx *Context) Respond(resp pdu.PDU, status pdu.Status) error {
	if resp == nil {
		return errors.New("smpp: responding with nil PDU")
	}

	ctx.status = status
	ctx.pdu = resp

    return ctx.Sess.SendResponse(ctx, resp, status)
}

func (ctx *Context) RespondWithSeq(resp pdu.PDU, seq uint32, status pdu.Status) error {
	if resp == nil {
		return errors.New("smpp: responding with nil PDU")
	}

	ctx.status = status
    ctx.seq = seq
	ctx.pdu = resp

    return ctx.Sess.SendResponse(ctx, resp, status)
}

func (ctx *Context) SendRequest(p pdu.PDU) (uint32, error) {
	return ctx.Sess.SendRequest(ctx.ctx, p)
}

func (ctx *Context) SendRequestWithSeq(p pdu.PDU, seq uint32) (uint32, error) {
	return ctx.Sess.SendRequest(ctx.ctx, p, pdu.EncodeSeq(seq))
}