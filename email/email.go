/*
Package email : email emtp发送模块
*/
package email

import (
	"crypto/tls"
	"errors"
	"sync"
	"time"

	gomail "gopkg.in/mail.v2"
)

// SMTPOpt smtp config
type SMTPOpt struct {
	// Username login username
	Username string
	// Passwd login password
	Passwd string
	// SMTPHost smtp server host
	SMTPHost string
	// SMTPPort smtp server port, default: 587
	SMTPPort int
}

// Data email body
type Data struct {
	// From sender name, if empty, use username
	From string
	// To target email address
	To string
	// Subject email title
	Subject string
	// Cc cc to other
	Cc string
	// Msg email body, default send as html
	Msg string
}

// EMail emal send client
type EMail struct {
	locker   *sync.Mutex
	dialer   *gomail.Dialer
	message  *gomail.Message
	username string
}

// NewEMail get a new email send client
func NewEMail(opt *SMTPOpt) (*EMail, error) {
	if opt == nil {
		return nil, errors.New("illegal smtp config")
	}
	if opt.SMTPHost == "" {
		return nil, errors.New("illegal smtp config")
	}
	if opt.SMTPPort < 1 || opt.SMTPPort > 65535 {
		opt.SMTPPort = 587
	}
	d := gomail.NewDialer(opt.SMTPHost, opt.SMTPPort, opt.Username, opt.Passwd)
	if d != nil {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		return &EMail{
			locker:   &sync.Mutex{},
			dialer:   d,
			username: opt.Username,
			message:  gomail.NewMessage(),
		}, nil
	}

	return nil, errors.New("unknow error")
}

// Send send email to target
func (e *EMail) Send(d *Data) error {
	if e == nil {
		return errors.New("email client is not ready")
	}
	if d == nil {
		return errors.New("nothing to send")
	}
	if d.To == "" {
		return errors.New("who do you want to send to?")
	}
	e.locker.Lock()
	defer e.locker.Unlock()
	if d.From == "" {
		d.From = e.username
	}
	if d.Subject == "" {
		d.Subject = "nil subject"
	}

	e.message.Reset()
	e.message.SetHeader("From", d.From)
	e.message.SetHeader("To", d.To)
	e.message.SetHeader("Subject", d.Subject)
	if len(d.Cc) > 0 {
		e.message.SetAddressHeader("Cc", d.Cc, d.Cc)
	}
	e.message.SetDateHeader("Date", time.Now())
	e.message.SetBody("text/html", d.Msg)
	return e.dialer.DialAndSend(e.message)
}
