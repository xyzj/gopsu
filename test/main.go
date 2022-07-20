package main

import (
	"github.com/xyzj/gopsu/email"
)

func main() {
	e, _ := email.NewEMail(&email.SMTPOpt{
		SMTPHost: "smtp.office365.com",
		Username: "minamoto.xu@hotmail.com",
		Passwd:   "Someone@140821",
	})
	e.Send(&email.Data{
		To:      "minamoto.xu@gmail.com",
		Subject: "cc test",
		Msg:     "cc to xuyuan",
		Cc:      "xuyuan8720@189.cn",
	})
	// doc, err := document.Open("document.docx")
	// if err != nil {
	// 	println(err.Error())
	// 	return
	// }
	// for _, p := range doc.Paragraphs() {
	// 	for _, r := range p.Runs() {
	// 		println(r.Text())
	// 	}
	// }
}
