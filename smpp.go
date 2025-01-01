// Package smpp implements SMPP protocol v3.4.
//
// It allows easier creation of SMPP clients and servers by providing utilities for PDU and session handling.
// In order to do any kind of interaction you first need to create an SMPP [Session](https://godoc.org/github.com/majiddarvishan/smpp#Session). Session is the main carrier of the protocol and enforcer of the specification rules.
//
// Naked session can be created with:
//
//	// You must provide already established connection and configuration struct.
//	sess := smpp.NewSession(conn, conf)
//
// But it's much more convenient to use helpers that would do the binding with the remote SMSC and return you session prepared for sending:
//
//	// Bind with remote server by providing config structs.
//	sess, err := smpp.BindTRx(sessConf, bindConf)
//
// And once you have the session it can be used for sending PDUs to the binded peer.
//
//	sm := smpp.SubmitSm{
//	    SourceAddr:      "11111111",
//	    DestinationAddr: "22222222",
//	    ShortMessage:    "Hello from SMPP!",
//	}
//	// Session can then be used for sending PDUs.
//	resp, err := sess.Send(p)
//
// Session that is no longer used must be closed:
//
//	sess.Close()
//
// If you want to handle incoming requests to the session specify SMPPHandler in session configuration when creating new session similarly to HTTPHandler from _net/http_ package:
//
//	conf := smpp.SessionConf{
//	    Handler: smpp.HandlerFunc(func(ctx *smpp.Context) {
//	        switch ctx.CommandID() {
//	        case pdu.UnbindID:
//	            ubd, err := ctx.Unbind()
//	            if err != nil {
//	                t.Errorf(err.Error())
//	            }
//	            resp := ubd.Response()
//	            if err := ctx.Respond(resp, pdu.StatusOK); err != nil {
//	                t.Errorf(err.Error())
//	            }
//	        }
//	    }),
//	}
//
// Detailed examples for SMPP client and server can be found in the examples dir.
package smpp

import (
	"context"
	"net"
	"time"

	"github.com/majiddarvishan/smpp/pdu"
)

const (
	// Version of the supported SMPP Protocol. Only supporting 3.4 for now.
	Version = 0x34
	// SequenceStart is the starting reference for sequence number.
	SequenceStart = 0x00000001
	// SequenceEnd s sequence number upper boundary.
	SequenceEnd = 0x7FFFFFFF
)

// BindConf is the configuration for binding to smpp servers.
type BindConf struct {
	// Bind will be attempted to this addr.
	Addr string
	// Mandatory fields for binding PDU.
	SystemID   string
	Password   string
	SystemType string
	AddrTon    int
	AddrNpi    int
	AddrRange  string
}

func bind(req pdu.PDU, sc SessionConf, bc BindConf) (*Session, error) {
	conn, err := net.Dial("tcp", bc.Addr)
	if err != nil {
		return nil, err
	}
	sess := NewSession(conn, sc)
	timeout := sc.WindowTimeout
	if timeout == 0 {
		timeout = time.Second * 5
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err = sess.SendRequest(ctx, req)
	if err != nil {
		return sess, err
	}
	return sess, nil
}

// BindTx binds transmitter session.
func BindTx(sc SessionConf, bc BindConf) (*Session, error) {
	return bind(&pdu.BindTx{
		SystemID:         bc.SystemID,
		Password:         bc.Password,
		SystemType:       bc.SystemType,
		InterfaceVersion: Version,
		AddrTon:          bc.AddrTon,
		AddrNpi:          bc.AddrNpi,
		AddressRange:     bc.AddrRange,
	}, sc, bc)
}

// BindRx binds receiver session.
func BindRx(sc SessionConf, bc BindConf) (*Session, error) {
	return bind(&pdu.BindRx{
		SystemID:         bc.SystemID,
		Password:         bc.Password,
		SystemType:       bc.SystemType,
		InterfaceVersion: Version,
		AddrTon:          bc.AddrTon,
		AddrNpi:          bc.AddrNpi,
		AddressRange:     bc.AddrRange,
	}, sc, bc)
}

// BindTRx binds transreceiver session.
func BindTRx(sc SessionConf, bc BindConf) (*Session, error) {
	return bind(&pdu.BindTRx{
		SystemID:         bc.SystemID,
		Password:         bc.Password,
		SystemType:       bc.SystemType,
		InterfaceVersion: Version,
		AddrTon:          bc.AddrTon,
		AddrNpi:          bc.AddrNpi,
		AddressRange:     bc.AddrRange,
	}, sc, bc)
}

// Unbind session will initiate session unbinding and close the session.
// First it will try to notify peer with unbind request.
// If there was any error during unbinding an error will be returned.
// Session will be closed even if there was an error during unbind.
func Unbind(ctx context.Context, sess *Session) error {
	defer func() {
		sess.Close()
	}()
	_, err := sess.SendRequest(ctx, pdu.Unbind{})
	if err != nil {
		return err
	}
	return nil
}

// SendGenericNack is a helper function for sending GenericNack PDU.
func SendGenericNack(ctx context.Context, sess *Session, p *pdu.GenericNack) error {
	_, err := sess.SendRequest(ctx, p)
	if err != nil {
		return err
	}
	return nil
}

// // SendBindRx is a helper function for sending BindRx PDU.
// func SendBindRx(ctx context.Context, sess *Session, p *pdu.BindRx) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendBindRxResp is a helper function for sending BindRxResp PDU.
// func SendBindRxResp(ctx context.Context, sess *Session, p *pdu.BindRxResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendBindTx is a helper function for sending BindTx PDU.
// func SendBindTx(ctx context.Context, sess *Session, p *pdu.BindTx) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendBindTxResp is a helper function for sending BindTxResp PDU.
// func SendBindTxResp(ctx context.Context, sess *Session, p *pdu.BindTxResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendBindTRx is a helper function for sending BindTRx PDU.
// func SendBindTRx(ctx context.Context, sess *Session, p *pdu.BindTRx) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendBindTRxResp is a helper function for sending BindTRxResp PDU.
// func SendBindTRxResp(ctx context.Context, sess *Session, p *pdu.BindTRxResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendUnbind is a helper function for sending Unbind PDU.
// func SendUnbind(ctx context.Context, sess *Session, p *pdu.Unbind) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendUnbindResp is a helper function for sending UnbindResp PDU.
// func SendUnbindResp(ctx context.Context, sess *Session, p *pdu.UnbindResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendSubmitSm is a helper function for sending SubmitSm PDU.
// func SendSubmitSm(ctx context.Context, sess *Session, p *pdu.SubmitSm) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendSubmitSmResp is a helper function for sending SubmitSmResp PDU.
// func SendSubmitSmResp(ctx context.Context, sess *Session, p *pdu.SubmitSmResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendDeliverSm is a helper function for sending DeliverSm PDU.
// func SendDeliverSm(ctx context.Context, sess *Session, p *pdu.DeliverSm) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendDeliverSmResp is a helper function for sending DeliverSmResp PDU.
// func SendDeliverSmResp(ctx context.Context, sess *Session, p *pdu.DeliverSmResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendOutbind is a helper function for sending Outbind PDU.
// func SendOutbind(ctx context.Context, sess *Session, p *pdu.Outbind) error {
// 	_, err := sess.SendRequest(ctx, p)
//     return err
// }

// // SendEnquireLink is a helper function for sending EnquireLink PDU.
// func SendEnquireLink(ctx context.Context, sess *Session, p *pdu.EnquireLink) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendEnquireLinkResp is a helper function for sending EnquireLinkResp PDU.
// func SendEnquireLinkResp(ctx context.Context, sess *Session, p *pdu.EnquireLinkResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendSubmitMulti is a helper function for sending SubmitMulti PDU.
// func SendSubmitMulti(ctx context.Context, sess *Session, p *pdu.SubmitMulti) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendSubmitMultiResp is a helper function for sending SubmitMultiResp PDU.
// func SendSubmitMultiResp(ctx context.Context, sess *Session, p *pdu.SubmitMultiResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendAlertNotification is a helper function for sending AlertNotification PDU.
// func SendAlertNotification(ctx context.Context, sess *Session, p *pdu.AlertNotification) error {
// 	_, err := sess.SendRequest(ctx, p)
//     return err
// }

// // SendDataSm is a helper function for sending DataSm PDU.
// func SendDataSm(ctx context.Context, sess *Session, p *pdu.DataSm) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendDataSmResp is a helper function for sending DataSmResp PDU.
// func SendDataSmResp(ctx context.Context, sess *Session, p *pdu.DataSmResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendQuerySm is a helper function for sending QuerySm PDU.
// func SendQuerySm(ctx context.Context, sess *Session, p *pdu.QuerySm) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendQuerySmResp is a helper function for sending QuerySmResp PDU.
// func SendQuerySmResp(ctx context.Context, sess *Session, p *pdu.QuerySmResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendReplaceSm is a helper function for sending ReplaceSm PDU.
// func SendReplaceSm(ctx context.Context, sess *Session, p *pdu.ReplaceSm) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendReplaceSmResp is a helper function for sending ReplaceSmResp PDU.
// func SendReplaceSmResp(ctx context.Context, sess *Session, p *pdu.ReplaceSmResp) error {
// 	err := sess.SendResponse(ctx, p)
//     return err
// }

// // SendCancelSm is a helper function for sending CancelSm PDU.
// func SendCancelSm(ctx context.Context, sess *Session, p *pdu.CancelSm) (uint32, error) {
// 	return sess.SendRequest(ctx, p)
// }

// // SendCancelSmResp is a helper function for sending CancelSmResp PDU.
// func SendCancelSmResp(ctx context.Context, sess *Session, p *pdu.CancelSmResp) error {
// 	err := sess.SendResponse(ctx, p, pdu.StatusOK)
//     return err
// }
