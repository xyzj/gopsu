package main

import (
	gomail "github.com/xyzj/gopsu/email"
)

func main() {
	m, err := gomail.NewEMail(&gomail.SMTPOpt{
		Username: "minamoto.xu@hotmail.com",
		Passwd:   "Someone@140821",
		SMTPHost: "smtp.office365.com",
	})
	if err != nil {
		println(err.Error())
		return
	}
	err = m.Send(&gomail.Data{
		To:      "xuyuan8720@189.cn",
		Subject: "Gomail test subject",
		Msg: `
	GO 发送邮件，官方连包都帮我们写好了，真是贴心啊！！！`,
	})
	if err != nil {
		println(err.Error())
		return
	}
	println("email send")
}
