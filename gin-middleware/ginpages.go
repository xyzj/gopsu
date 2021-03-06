package ginmiddleware

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
	"github.com/xyzj/gopsu"
)

var (
	template500        = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>devQuotes</title><meta name="viewport" content="width=device-width, initial-scale=1"><style>* {margin:0;padding:0;border:0;font-size:100%;font:inherit;vertical-align:baseline;box-sizing:border-box;color:inherit}body {background-image:linear-gradient(120deg, #4f0088 0%, #000000 100%);height:100vh}h1 {font-size:15vw;text-align:center;position:fixed;width:100vw;z-index:1;color:#ffffff26;text-shadow:0 0 50px rgba(0, 0, 0, 0.07);top:50%;-webkit-transform:translateY(-50%);transform:translateY(-50%);font-family:"Montserrat", monospace}div {background:rgba(0, 0, 0, 0);width:70vw;position:relative;top:50%;-webkit-transform:translateY(-50%);transform:translateY(-50%);margin:0 auto;padding:30px 30px 10px;box-shadow:0 0 150px-20px rgba(0, 0, 0, 0.5);z-index:3}P {font-family:"Share Tech Mono", monospace;color:#f5f5f5;margin:0 0 20px;font-size:17px;line-height:1.2}span {color:#f0c674;font-size:5vw}i {color:#8abeb7;font-size:3vw}div a {text-decoration:none}b {color:#81a2be}a.avatar {position:fixed;bottom:15px;right:-100px;-webkit-animation:slide 10.5s 40.5s forwards;animation:slide 10.5s 40.5s forwards;display:block;z-index:4}a.avatar img {border-radius:100%;width:44px;border:2px solid white}@-webkit-keyframes slide {from {right:-100px;-webkit-transform:rotate(360deg);transform:rotate(360deg);opacity:0}to {right:15px;-webkit-transform:rotate(0deg);transform:rotate(0deg);opacity:1}}@keyframes slide {from {right:-100px;-webkit-transform:rotate(360deg);transform:rotate(360deg);opacity:0}to {right:15px;-webkit-transform:rotate(0deg);transform:rotate(0deg);opacity:1}}</style></head><body><h1>devQuotes</h1><div><p><i>I don't always test my code, but when I do, I do it in production.</i></p></br></br></br><p><span>GOOD LUCK !!!</span></p></div><script>var str = document.getElementsByTagName('div')[0].innerHTML.toString(); var i = 0; document.getElementsByTagName('div')[0].innerHTML = ""; setTimeout(function () { var se = setInterval(function () { i++; document.getElementsByTagName('div')[0].innerHTML = str.slice(0, i) + "|"; if (i == str.length) { clearInterval(se); document.getElementsByTagName('div')[0].innerHTML = str } }, 10) }, 0);</script></body></html>`
	template403City    = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>403🐲Forbidden City</title><style>@import url("https://fonts.googleapis.com/css?family=Permanent+Marker");@import url("https://fonts.googleapis.com/css?family=Roboto+Mono");html,body{width:100%;height:100%;margin:0;padding:0}body{background:#F3E2CB;display:-webkit-box;display:flex;-webkit-box-orient:vertical;-webkit-box-direction:normal;flex-direction:column;-webkit-box-align:center;align-items:center;align-content:center}.wrapper{height:100%;width:100%;display:-webkit-box;display:flex;-webkit-box-orient:vertical;-webkit-box-direction:reverse;flex-direction:column-reverse;-webkit-box-align:center;align-items:center;align-content:center;position:absolute;bottom:0;overflow:hidden}.wrapper:hover.sun{-webkit-transform:translateY(-200px);transform:translateY(-200px)}.pedastal{width:1000px;height:90px;background:white;position:relative}.pedastal-block1,.pedastal-block1::before{width:125px;height:30px;background:#A24D4C;box-sizing:border-box}.pedastal-block1::before{content:'';position:absolute;right:0}.pedastal-block2,.pedastal-block2::before{width:63px;height:30px;background:#A24D4C;box-sizing:border-box}.pedastal-block2::before{content:'';position:absolute;right:0}.hall{width:520px;height:60px;background:#44291E;display:-webkit-box;display:flex;-webkit-box-orient:horizontal;-webkit-box-direction:normal;flex-direction:row;-webkit-box-pack:justify;justify-content:space-between;position:relative;z-index:3}.hall-pillar{height:100%;width:16px;background:-webkit-gradient(linear,left top,right top,from(#DA5447),to(#9C4E46));background:linear-gradient(90deg,#DA5447,#9C4E46)}.hall-support{width:40px;height:12px;position:relative}.hall-support::before{content:'';width:16px;height:12px;background:linear-gradient(135deg,#678B80 50%,transparent 51%)no-repeat;background-position:-2px 0;position:absolute;top:0;left:0}.hall-support::after{content:'';width:16px;height:12px;background:linear-gradient(-135deg,#678B80 50%,transparent 51%)no-repeat;background-position:2px 0;position:absolute;top:0;right:0}.lower-support{width:520px;height:30px;background:#7BA598;display:-webkit-box;display:flex;-webkit-box-orient:horizontal;-webkit-box-direction:normal;flex-direction:row;-webkit-box-pack:justify;justify-content:space-between;position:relative;border-left:4px solid#7BA598;border-right:4px solid#7BA598;z-index:3}.lower-support-pillar{height:100%;width:16px;background:-webkit-gradient(linear,left top,right top,from(#87C9B6),to(#678B80));background:linear-gradient(90deg,#87C9B6,#678B80)}.ornaments{width:40px;height:30px;display:-webkit-box;display:flex}.ornaments div{width:20px;height:30px;position:relative}.ornaments div:first-child::before,.ornaments div:first-child::after{content:'';width:8px;height:8px;border-radius:4px;background:#EEDB44;position:absolute}.ornaments div:first-child::before{top:6px;left:11px}.ornaments div:first-child::after{bottom:6px;left:11px}.ornaments div:last-child::before,.ornaments div:last-child::after{content:'';width:8px;height:8px;border-radius:4px;background:#EEDB44;position:absolute}.ornaments div:last-child::before{top:6px;right:11px}.ornaments div:last-child::after{bottom:6px;right:11px}.lower-roof{width:376px;height:40px;background:#FDBB3B;position:relative;z-index:3}.lower-roof::before{content:'';border-left:112px solid transparent;border-bottom:40px solid#FDBB3B;position:absolute;bottom:0;left:-112px}.lower-roof::after{content:'';border-right:112px solid transparent;border-bottom:40px solid#FDBB3B;position:absolute;bottom:0;right:-112px}.lower-roof div:first-child{display:inline-block;border-left:36px solid transparent;border-top:15px solid#D0982E;position:absolute;left:-112px;bottom:-15px}.lower-roof div:last-child{display:inline-block;border-right:36px solid transparent;border-top:15px solid#D0982E;position:absolute;right:-112px;bottom:-15px}.upper-support{width:376px;height:20px;background:#7BA598;display:-webkit-box;display:flex;-webkit-box-orient:horizontal;-webkit-box-direction:normal;flex-direction:row;-webkit-box-pack:justify;justify-content:space-between;position:relative;z-index:3}.upper-support.container{width:296px;height:20px;display:-webkit-box;display:flex;align-self:center;-webkit-box-orient:horizontal;-webkit-box-direction:normal;flex-direction:row;-webkit-box-pack:justify;justify-content:space-between;position:absolute;left:50%;-webkit-transform:translateX(-50%);transform:translateX(-50%)}.upper-support.ornaments div:first-child::after,.upper-support.ornaments div:last-child::after{display:none}.upper-support.ornaments div:first-child::before,.upper-support.ornaments div:last-child::before{width:6px;height:6px}.upper-support.ornaments div:first-child::before{top:7px;left:8px}.upper-support.ornaments div:last-child::before{top:7px;right:8px}.upper-roof{width:520px;height:90px;position:relative;z-index:3}.upper-roof div:first-child{display:inline-block;border-left:72px solid transparent;border-top:20px solid#D0982E;position:absolute;left:0px;bottom:-20px}.upper-roof div:last-child{display:inline-block;border-right:72px solid transparent;border-top:20px solid#D0982E;position:absolute;right:0px;bottom:-20px}.upper-roof-curved{width:100px;height:78px;background:#F3E2CB;position:absolute;z-index:1000}.upper-roof-curved:nth-child(2){left:-102px;top:-2px;-webkit-transform:rotate(3deg);transform:rotate(3deg);border-radius:0 0 100px 0}.upper-roof-curved:nth-child(3){right:-102px;top:-2px;-webkit-transform:rotate(-3deg);transform:rotate(-3deg);border-radius:0 0 0 100px}.roof-top div,.roof-top div:first-child::before,.roof-top div:first-child::after,.roof-top div:last-child::before,.roof-top div:last-child::after{width:8px;height:8px;background:#FDBB3B;position:absolute}.roof-top{width:264px;position:relative;z-index:3}.roof-top div{top:-8px}.roof-top div:first-child{left:0px}.roof-top div:first-child::before,.roof-top div:first-child::after{content:'';left:8px}.roof-top div:first-child::after{bottom:8px}.roof-top div:last-child{right:0px}.roof-top div:last-child::before,.roof-top div:last-child::after{content:'';right:8px}.roof-top div:last-child::after{bottom:8px}.sign{width:12px;height:16px;background:#490CED;border:4px solid#9C4E46;position:absolute;left:50%;-webkit-transform:translateX(-50%);transform:translateX(-50%)}.trapezium{border-bottom:90px solid#F8DAB2;border-right:50px solid transparent;border-left:50px solid transparent;width:288px;position:absolute;bottom:0;left:50%;-webkit-transform:translateX(-50%);transform:translateX(-50%)}.trapezium div{position:absolute;bottom:-90px;width:20px}.trapezium div::before{content:'';position:absolute;width:20px}.trapezium div:first-child{left:24px;border-bottom:90px solid white;border-left:40px solid transparent}.trapezium div:first-child::before{border-top:90px solid white;border-right:40px solid transparent}.trapezium div:last-child{right:24px;border-bottom:90px solid white;border-right:40px solid transparent}.trapezium div:last-child::before{border-top:90px solid white;border-left:40px solid transparent;right:0}.wall{width:100%;height:90px;background:#A24D4C;position:fixed;bottom:0;z-index:-1;display:-webkit-box;display:flex;-webkit-box-pack:center;justify-content:center}.wall::before{content:'';width:100%;max-width:1240px;height:140px;background:#A24D4C;position:absolute;bottom:0;left:50%;-webkit-transform:translateX(-50%);transform:translateX(-50%)}.wall-roofing-bottom{width:100%;height:24px;background:#FDBB3B}.wall-roofing-top{width:1240px;height:24px;background:#FDBB3B;position:absolute;top:-50px}.wall-roofing-top::before{content:'';border-bottom:24px solid#FDBB3B;border-left:10px solid transparent;position:absolute;left:-10px}.wall-roofing-top::after{content:'';border-bottom:24px solid#FDBB3B;border-right:10px solid transparent;position:absolute;right:-10px}.sun{width:400px;height:400px;background:#CA502E;border-radius:200px;z-index:1;position:absolute;-webkit-transform:translateY(-100px);transform:translateY(-100px);display:-webkit-box;display:flex;-webkit-box-pack:center;justify-content:center;-webkit-box-align:center;align-items:center;-webkit-transition:-webkit-transform 1s;transition:-webkit-transform 1s;transition:transform 1s;transition:transform 1s,-webkit-transform 1s}.cloud{background:white;position:relative;z-index:2}.cloud::before,.cloud::after{background:white;display:block}.cloud-01{width:88px;height:32px;border-radius:16px;-webkit-transform:translate(-200px,-50px);transform:translate(-200px,-50px);-webkit-animation:cloud-1 50s ease-in-out infinite alternate;animation:cloud-1 50s ease-in-out infinite alternate}.cloud-01::before{content:'';width:50px;height:50px;border-radius:25px;display:block;-webkit-transform:translate(22px,-25px);transform:translate(22px,-25px)}.cloud-02{width:100px;height:40px;border-radius:20px;-webkit-transform:translate(60px,-120px);transform:translate(60px,-120px);-webkit-animation:cloud-2 40s ease-in-out infinite alternate;animation:cloud-2 40s ease-in-out infinite alternate}.cloud-02::before{content:'';width:46px;height:46px;border-radius:23px;-webkit-transform:translate(38px,-23px);transform:translate(38px,-23px)}.cloud-02::after{content:'';width:30px;height:30px;border-radius:15px;-webkit-transform:translate(16px,-60px);transform:translate(16px,-60px)}.cloud-03{width:70px;height:24px;border-radius:12px;-webkit-transform:translate(210px,0px);transform:translate(210px,0px);-webkit-animation:cloud-3 30s ease-in-out infinite alternate;animation:cloud-3 30s ease-in-out infinite alternate}.cloud-03::before{content:'';width:14px;height:14px;border-radius:7px;-webkit-transform:translate(46px,-7px);transform:translate(46px,-7px)}.cloud-03::after{content:'';width:16px;height:16px;border-radius:8px;top:0;-webkit-transform:translate(12px,-50px);transform:translate(12px,-50px)}.cloud-03 div{width:30px;height:30px;background:white;border-radius:15px;display:block;-webkit-transform:translate(24px,-30px);transform:translate(24px,-30px)}.copy{font-family:'Permanent Marker',cursive;font-size:8em;color:#F3E2CB;padding-bottom:60px}.headline{text-align:center;position:relative;padding-top:40px;z-index:3}.headline h1{font-family:'Permanent Marker',cursive;color:#2b2b2b;font-size:8em;margin:0}.headline h2{font-family:'Roboto Mono',monospace;font-size:1.25em;color:#2b2b2b}@-webkit-keyframes cloud-1{0%{-webkit-transform:translate(-200px,-50px);transform:translate(-200px,-50px)}100%{-webkit-transform:translate(-280px,-50px);transform:translate(-280px,-50px)}}@keyframes cloud-1{0%{-webkit-transform:translate(-200px,-50px);transform:translate(-200px,-50px)}100%{-webkit-transform:translate(-280px,-50px);transform:translate(-280px,-50px)}}@-webkit-keyframes cloud-2{0%{-webkit-transform:translate(60px,-120px);transform:translate(60px,-120px)}100%{-webkit-transform:translate(300px,-120px);transform:translate(300px,-120px)}}@keyframes cloud-2{0%{-webkit-transform:translate(60px,-120px);transform:translate(60px,-120px)}100%{-webkit-transform:translate(300px,-120px);transform:translate(300px,-120px)}}@-webkit-keyframes cloud-3{0%{-webkit-transform:translate(210px,0px);transform:translate(210px,0px)}100%{-webkit-transform:translate(100px,0px);transform:translate(100px,0px)}}@keyframes cloud-3{0%{-webkit-transform:translate(210px,0px);transform:translate(210px,0px)}100%{-webkit-transform:translate(100px,0px);transform:translate(100px,0px)}}@media only screen and(max-width:1440px){.headline h1{font-size:4em}.headline h2{font-size:1em}}</style></head><body><!--partial:index.partial.html--><div class="wrapper"><section class="pedastal"><div class="pedastal-block1"></div><div class="pedastal-block2"></div><div class="trapezium"><div></div><div></div></div></section><section class="hall"><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div><div class="hall-support"></div><div class="hall-pillar"></div></section><section class="lower-support"><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div><div class="ornaments"><div></div><div></div></div><div class="lower-support-pillar"></div></section><section class="lower-roof"><div></div><div></div></section><section class="upper-support"><div class="container"><div class="lower-support-pillar"></div><div class="lower-support-pillar"></div><div class="lower-support-pillar"></div><div class="lower-support-pillar"></div><div class="lower-support-pillar"></div><div class="lower-support-pillar"></div></div><div class="ornaments"><div></div><div></div></div><div class="ornaments"><div></div><div></div></div><div class="ornaments"><div></div><div></div></div><div class="ornaments"><div></div><div></div></div><div class="ornaments"><div></div><div></div></div><div class="ornaments"><div></div><div></div></div><div class="ornaments"><div></div><div></div></div><section class="sign"></section></section><section class="upper-roof"><div></div><svg width="520px"height="90px"><path d="M495.689265,72.9065145 L520,90 L0,90 L24.3069308,72.9091893 L45.9698498,74.0444947 C88.9890231,76.2990341 125.690619,43.2527689 127.945158,0.233595624 L127.687016,0.220066965 L128,0 L392,0 L392.30918,0.217392187 L392,0.233595624 C394.254539,43.2527689 430.956135,76.2990341 473.975308,74.0444947 L495.689265,72.9065145 Z"id="Combined-Shape"fill="#FDBB3B"></path></svg><div></div></section><section class="roof-top"><div></div><div></div></section><div class="cloud cloud-01"></div><div class="cloud cloud-02"></div><div class="cloud cloud-03"><div></div></div><div class="sun"><div class="copy">403</div></div></div><div class="wall"><div class="wall-roofing-bottom"></div><div class="wall-roofing-top"></div></div><div class="headline"><h1>Forbidden City</h1><h2>✋You don't have permission to access🚧this area✋</h2></div><!--partial--></body></html>`
	template403        = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>403 Forbidden Console</title><meta name="viewport"content="width=device-width, initial-scale=1"><style>@import url("https://fonts.googleapis.com/css?family=Share+Tech+Mono|Montserrat:700");*{margin:0;padding:0;border:0;font-size:100%;font:inherit;vertical-align:baseline;box-sizing:border-box;color:inherit}body{background-image:linear-gradient(120deg,#4f0088 0%,#000000 100%);height:100vh}h1{font-size:45vw;text-align:center;position:fixed;width:100vw;z-index:1;color:#ffffff26;text-shadow:0 0 50px rgba(0,0,0,0.07);top:50%;-webkit-transform:translateY(-50%);transform:translateY(-50%);font-family:"Montserrat",monospace}div{background:rgba(0,0,0,0);width:70vw;position:relative;top:50%;-webkit-transform:translateY(-50%);transform:translateY(-50%);margin:0 auto;padding:30px 30px 10px;box-shadow:0 0 150px-20px rgba(0,0,0,0.5);z-index:3}P{font-family:"Share Tech Mono",monospace;color:#f5f5f5;margin:0 0 20px;font-size:17px;line-height:1.2}span{color:#f0c674}i{color:#8abeb7}div a{text-decoration:none}b{color:#81a2be}a.avatar{position:fixed;bottom:15px;right:-100px;-webkit-animation:slide 0.5s 4.5s forwards;animation:slide 0.5s 4.5s forwards;display:block;z-index:4}a.avatar img{border-radius:100%;width:44px;border:2px solid white}@-webkit-keyframes slide{from{right:-100px;-webkit-transform:rotate(360deg);transform:rotate(360deg);opacity:0}to{right:15px;-webkit-transform:rotate(0deg);transform:rotate(0deg);opacity:1}}@keyframes slide{from{right:-100px;-webkit-transform:rotate(360deg);transform:rotate(360deg);opacity:0}to{right:15px;-webkit-transform:rotate(0deg);transform:rotate(0deg);opacity:1}}</style></head><body><!--partial:index.partial.html--><h1>403</h1><div><p>><span>ERROR CODE</span>:"<i>HTTP 403 Forbidden</i>"</p><p>><span>ERROR DESCRIPTION</span>:"<i>Access Denied. You Do Not Have The Permission To Access Here On This Server</i>"</p><p>><span>ERROR POSSIBLY CAUSED BY</span>:[<b>execute access forbidden,read access forbidden,write access forbidden,ssl required,ssl 128 required,ip address rejected,client certificate required,site access denied,too many users,invalid configuration,password change,mapper denied access,client certificate revoked,directory listing denied,client access licenses exceeded,client certificate is untrusted or invalid,client certificate has expired or is not yet valid,passport logon failed,source access denied,infinite depth is denied,too many requests from the same client ip</b>...]</p><p>><span>SOME PAGES ON THIS SERVER THAT YOU DO HAVE PERMISSION TO ACCESS</span>:[<a href="/">Home Page</a>]</p><p>><span>HAVE A NICE DAY SIR:-)</span></p></div><!--partial--><script>var str=document.getElementsByTagName('div')[0].innerHTML.toString();var i=0;document.getElementsByTagName('div')[0].innerHTML="";setTimeout(function(){var se=setInterval(function(){i++;document.getElementsByTagName('div')[0].innerHTML=str.slice(0,i)+"|";if(i==str.length){clearInterval(se);document.getElementsByTagName('div')[0].innerHTML=str}},10)},0);</script></body></html>`
	template404        = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>404</title><style>@import url(https://fonts.googleapis.com/css?family=Exo+2:200i);:root{font-size:10px;--neon-text-color:#f40;--neon-border-color:#08f}body{display:flex;margin:0;padding:0;min-height:100vh;border:0;background:#000;font-size:100%;font-family:'Exo 2',sans-serif;line-height:1;justify-content:center;align-items:center}h1{padding:4rem 6rem 5.5rem;border:.4rem solid #fff;border-radius:2rem;color:#fff;text-transform:uppercase;font-weight:200;font-style:italic;font-size:13rem;animation:flicker 1s infinite alternate}h1::-moz-selection{background-color:var(--neon-border-color);color:var(--neon-text-color)}h1::selection{background-color:var(--neon-border-color);color:var(--neon-text-color)}h1:focus{outline:0}.flicker-text-fast{animation:flicker-text 1.5s infinite alternate}.flicker-text-slow{animation:flicker-text 4.4s infinite alternate}@keyframes flicker{0%{box-shadow:0 0 .5rem #fff,inset 0 0 .5rem #fff,0 0 2rem var(--neon-border-color),inset 0 0 2rem var(--neon-border-color),0 0 4rem var(--neon-border-color),inset 0 0 4rem var(--neon-border-color);text-shadow:-.2rem -.2rem 1rem #fff,.2rem .2rem 1rem #fff,0 0 2rem var(--neon-text-color),0 0 4rem var(--neon-text-color),0 0 6rem var(--neon-text-color),0 0 8rem var(--neon-text-color),0 0 10rem var(--neon-text-color)}100%{box-shadow:0 0 .5rem #fff,inset 0 0 .5rem #fff,0 0 2rem var(--neon-border-color),inset 0 0 2rem var(--neon-border-color),0 0 4rem var(--neon-border-color),inset 0 0 4rem var(--neon-border-color);text-shadow:-.2rem -.2rem 1rem #fff,.2rem .2rem 1rem #fff,0 0 2rem var(--neon-text-color),0 0 4rem var(--neon-text-color),0 0 6rem var(--neon-text-color),0 0 8rem var(--neon-text-color),0 0 10rem var(--neon-text-color)}}@keyframes flicker-text{0%,100%,19%,21%,23%,25%,54%,56%{box-shadow:none;text-shadow:-.2rem -.2rem 1rem #fff,.2rem .2rem 1rem #fff,0 0 2rem var(--neon-text-color),0 0 4rem var(--neon-text-color),0 0 6rem var(--neon-text-color),0 0 8rem var(--neon-text-color),0 0 10rem var(--neon-text-color)}20%,24%,55%{box-shadow:none;text-shadow:none}}</style></head><body><h1 contenteditable spellcheck="false"><span class="flicker-text-slow">40</span><span class="flicker-text-fast">4</span></h1></body></html>`
	templateHelloWorld = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>Aloha</title><style>@import url(https://fonts.googleapis.com/css?family=Montserrat:700);body{margin:0;width:100%;height:100vh;overflow:hidden;background:hsla(0,5%,5%,1);background-repeat:no-repeat;background-attachment:fixed;background-image:-webkit-gradient(linear,left bottom,right top,from(hsla(0,5%,15%,0.5)),to(hsla(0,5%,5%,1)));background-image:linear-gradient(to right top,hsla(0,5%,15%,0.5),hsla(0,5%,5%,1))}svg{width:100%}</style></head><body><svg width="100%" height="100%" viewBox="30 -50 600 500" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" version="1.1"><path id="path"><animate attributeName="d" from="m0,110 h0" to="m0,110 h1100" dur="7s" begin="0.5s" repeatCount="indefinite"/></path><text font-size="30" font-family="Montserrat" fill='hsla(36, 95%, 85%, 1)'><textPath xlink:href="#path">Aloha! You are good to Go 🍹</textPath></text></svg></body></html>`
	templateRuntime    = `<html lang="zh-cn">
<head>
    <meta content="text/html; charset=utf-8" http-equiv="content-type" />
    <!-- <script language="JavaScript">
			function myrefresh(){window.location.reload()}
	    	setTimeout('myrefresh()',180000); //指定180s刷新一次
	    </script> -->
    <style type="text/css">a{color:#4183C4;font-size:16px;}h1,h2,h3,h4,h5,h6{margin:20px 0 10px;padding:0;font-weight:bold;-webkit-font-smoothing:antialiased;cursor:text;position:relative;}h1{font-size:28px;color:black;}h2{font-size:24px;border-bottom:1px solid #cccccc;color:black;}h3{font-size:18px;}h4{font-size:16px;}h5{font-size:14px;}h6{color:#777777;font-size:14px;}table{padding:0;}table tr{border-top:1px solid #cccccc;background-color:white;margin:0;padding:0;}table tr:nth-child(2n){background-color:#f8f8f8;}table tr th{font-weight:bold;border:1px solid #cccccc;text-align:center;margin:0;padding:6px 13px;}table tr td{border:1px solid #cccccc;text-align:center;margin:0;padding:6px 13px;}table tr th:first-child,table tr td:first-child{margin-top:0;}table tr th:last-child,table tr td:last-child{margin-bottom:0;}</style>
</head>

<body>
    <h3>服务器时间：</h3><a>{{.timer}}</a>
    <h3>{{.key}}：</h3><a>{{range $idx, $elem := .value}}
        {{$elem}} <br>
        {{end}}</a>
</body>
</html>`
)

var (
	runtimeInfo map[string]interface{}
)

func init() {
	runtimeInfo = make(map[string]interface{})
	runtimeInfo["timer"] = time.Now().Format("2006-01-02 15:04:05 Mon")
	runtimeInfo["value"] = []string{}
}

// GetTemplateRuntime 返回runtime模板
func GetTemplateRuntime() string {
	return templateRuntime
}

// Page403 Page403
func Page403(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(200, template403)
}

// Page404 Page404
func Page404(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(200, template404)
}

// Page405 Page405
func Page405(c *gin.Context) {
	c.String(http.StatusMethodNotAllowed, "method not allowed")
}

// Page500 Page500
func Page500(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(200, template500)
}

// PageDefault 健康检查
func PageDefault(c *gin.Context) {
	switch c.Request.Method {
	case "GET":
		if c.Request.RequestURI == "/" {
			c.Header("Content-Type", "text/html")
			c.String(200, templateHelloWorld)
		} else {
			c.String(200, "ok")
		}
	case "POST":
		c.String(200, "ok")
	}
}

// PageRuntime 启动信息显示
func PageRuntime(c *gin.Context) {
	if len(runtimeInfo["value"].([]string)) == 0 {
		_, fn, _, ok := runtime.Caller(0)
		if ok {
			b, err := ioutil.ReadFile(path.Base(fn) + ".ver")
			if err == nil {
				runtimeInfo["value"] = strings.Split(string(b), "\n")
			}
		}
	}
	runtimeInfo["time"] = time.Now().Format("2006-01-02 15:04:05 Mon")
	runtimeInfo["key"] = "服务运行信息"
	switch c.Request.Method {
	case "GET":
		c.Header("Content-Type", "text/html")
		// c.Status(http.StatusOK)
		// render.WriteString(c.Writer, templateRuntime, nil)
		t, _ := template.New("runtime").Parse(templateRuntime)
		h := render.HTML{
			Name:     "runtime",
			Data:     runtimeInfo,
			Template: t,
		}
		h.WriteContentType(c.Writer)
		h.Render(c.Writer)
	case "POST":
		c.PureJSON(200, runtimeInfo)
	}
}

// Clearlog 日志清理
func Clearlog(c *gin.Context) {
	if c.Param("pwd") != "xyissogood" {
		c.String(200, "Wrong!!!")
		return
	}
	var days int64
	if days = gopsu.String2Int64(c.Param("days"), 0); days == 0 {
		days = 7
	}
	// 遍历文件夹
	dir := c.Param("dir")
	if dir == "" {
		dir = gopsu.DefaultLogDir
	}
	lstfno, ex := ioutil.ReadDir(dir)
	if ex != nil {
		ioutil.WriteFile("ginlogerr.log", []byte(fmt.Sprintf("clear log files error: %s", ex.Error())), 0664)
	}
	t := time.Now()
	for _, fno := range lstfno {
		if fno.IsDir() || !strings.Contains(fno.Name(), c.Param("name")) { // 忽略目录，不含日志名的文件，以及当前文件
			continue
		}
		// 比对文件生存期
		if t.Unix()-fno.ModTime().Unix() >= days*24*60*60-10 {
			os.Remove(filepath.Join(c.Param("dir"), fno.Name()))
			c.Set(fno.Name(), "deleted")
		}
	}
	c.PureJSON(200, c.Keys)
}

// SetVersionInfo 设置服务版本信息
func SetVersionInfo(ver string) {
	runtimeInfo["ver"] = strings.Split(ver, "\n")[1:]
}
