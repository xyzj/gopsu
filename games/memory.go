package games

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

var (
	tplmemory = `<!DOCTYPE html>
<html lang="en">

<head>
	<meta charset="UTF-8">
	<title>CodePen - Tangram Memory Game</title>
	<link href="https://fonts.googleapis.com/css?family=Ribeye+Marrow|Montserrat:300" rel="stylesheet">
	<style>button,hr,input{overflow:visible}audio,canvas,progress,video{display:inline-block}progress,sub,sup{vertical-align:baseline}html{font-family:sans-serif;line-height:1.15;-ms-text-size-adjust:100%;-webkit-text-size-adjust:100%}body{margin:0} menu,article,aside,details,footer,header,nav,section{display:block}h1{font-size:2em;margin:.67em 0}figcaption,figure,main{display:block}figure{margin:1em 40px}hr{box-sizing:content-box;height:0}code,kbd,pre,samp{font-family:monospace,monospace;font-size:1em}a{background-color:transparent;-webkit-text-decoration-skip:objects}a:active,a:hover{outline-width:0}abbr[title]{border-bottom:none;text-decoration:underline;text-decoration:underline dotted}b,strong{font-weight:bolder}dfn{font-style:italic}mark{background-color:#ff0;color:#000}small{font-size:80%}sub,sup{font-size:75%;line-height:0;position:relative}sub{bottom:-.25em}sup{top:-.5em}audio:not([controls]){display:none;height:0}img{border-style:none}svg:not(:root){overflow:hidden}button,input,optgroup,select,textarea{font-family:sans-serif;font-size:100%;line-height:1.15;margin:0}button,input{}button,select{text-transform:none}[type=submit], [type=reset],button,html [type=button]{-webkit-appearance:button}[type=button]::-moz-focus-inner,[type=reset]::-moz-focus-inner,[type=submit]::-moz-focus-inner,button::-moz-focus-inner{border-style:none;padding:0}[type=button]:-moz-focusring,[type=reset]:-moz-focusring,[type=submit]:-moz-focusring,button:-moz-focusring{outline:ButtonText dotted 1px}fieldset{border:1px solid silver;margin:0 2px;padding:.35em .625em .75em}legend{box-sizing:border-box;color:inherit;display:table;max-width:100%;padding:0;white-space:normal}progress{}textarea{overflow:auto}[type=checkbox],[type=radio]{box-sizing:border-box;padding:0}[type=number]::-webkit-inner-spin-button,[type=number]::-webkit-outer-spin-button{height:auto}[type=search]{-webkit-appearance:textfield;outline-offset:-2px}[type=search]::-webkit-search-cancel-button,[type=search]::-webkit-search-decoration{-webkit-appearance:none}::-webkit-file-upload-button{-webkit-appearance:button;font:inherit}summary{display:list-item}[hidden],template{display:none}</style>
	<style>*,*:before,*:after{box-sizing:border-box}body{padding:1rem;min-height:100vh;border:1rem solid;background:#926ea9;color:wheat;text-align:center}h1{margin:0;font-weight:400;font-family:"Ribeye Marrow",sans-serif}p{font-family:Montserrat,sans-serif;font-weight:300;margin:0 0 1.5rem}.t svg{display:block}.t use{transition:.5s}.t:not(.revealed) use{fill:#78558f}use{stroke:wheat}.revealed use{stroke:transparent}.board{max-width:75vh;margin:0 auto;display:grid;grid-gap:.5rem;grid-template-columns:repeat(4,1fr);grid-auto-rows:1fr}.t{border:1px solid}.revealed{background:white;border-color:white}.t .bt-1{transform:translate(0,0)}.t .bt-2{transform:translate(0,0) scale(-1,1) rotate(90deg)}.t .st-1{transform:translate(25px,75px) scale(0.5,0.5) rotate(-90deg)}.t .mt{transform:translate(100px,49.5px) scale(0.7142,0.7142) rotate(45deg)}.t .st-2{transform:translate(100px,50px) scale(0.5,0.5) rotate(-180deg)}.tcat.revealed svg{transform:translate(0,8px)}.tcat.revealed .bt-1{transform:translate(12.5px,10px);fill:#3d3d3d}.tcat.revealed .bt-2{transform:translate(-8.5px,130px) scale(-1,1) rotate(135deg);fill:#666}.tcat.revealed .st-1{transform:translate(25px,0px) scale(0.5,0.5) rotate(180deg)}.tcat.revealed .st-2{transform:translate(-25px,-50px) scale(0.5,0.5) rotate(0deg)}.tcat.revealed .st-1,.tcat.revealed .st-2{fill:#ffe6c5}.tcat.revealed .mt{transform:translate(12.5px,12.5px) scale(-0.7142,0.7142) rotate(0deg)}.tcat.revealed .mt,.tcat.revealed .rh{fill:#ccc}.tcat.revealed .sq{transform:translate(-75px,-50px);fill:#303030}.tcat.revealed .rh{transform:translate(133px,200px) scale(1,-1) rotate(45deg)}.tsquirrel.revealed svg{transform:translate(0,12px)}.tsquirrel.revealed .bt-1{transform:translate(12.5px,-12.5px);fill:#6b3010}.tsquirrel.revealed .bt-2{transform:translate(-8.5px,108.5px) scale(-1,1) rotate(135deg);fill:#7f381c}.tsquirrel.revealed .st-1{transform:translate(12.5px,87.5px) scale(0.5,0.5) rotate(180deg)}.tsquirrel.revealed .st-2{transform:translate(97.5px,108.5px) scale(0.5,0.5) rotate(135deg)}.tsquirrel.revealed .mt{transform:translate(12.5px,-35px) scale(0.7142,0.7142) rotate(45deg)}.tsquirrel.revealed .sq{transform:translate(22.5px,-16.5px);fill:#e6e6e6}.tsquirrel.revealed .rh{transform:translate(172.5px,33.25px) scale(1,1) rotate(90deg);fill:#7f381c}.tsquirrel.revealed .st-1,.tsquirrel.revealed .st-2,.tsquirrel.revealed .mt{fill:#8c6239}.theart.revealed .bt-1{transform:translate(-25px,50px) rotate(-90deg);fill:#d4145a}.theart.revealed .bt-2{transform:translate(25px,50px) scale(-1,1) rotate(90deg);fill:#c1272d}.theart.revealed .st-1{transform:translate(50px,25px) scale(0.5,0.5) rotate(-90deg);fill:#9e005d}.theart.revealed .st-2{transform:translate(25px,50px) scale(0.5,0.5) rotate(0deg);fill:#d91119}.theart.revealed .mt{transform:translate(-25px,50px) scale(0.7142,0.7142) rotate(-45deg);fill:#ed1c24}.theart.revealed .sq{transform:translate(-25px,50px);fill:#c5282e}.theart.revealed .rh{transform:translate(50px,125px) scale(1,-1) rotate(0deg);fill:#d4145a}.trocket.revealed .bt-1{transform:translate(75px,75px) rotate(-180deg);fill:#ed1c24}.trocket.revealed .bt-2{transform:translate(75px,75px) scale(-1,1) rotate(180deg);fill:#c1272d}.trocket.revealed .st-1{transform:translate(-37.5px,62.5px) scale(0.5,0.5) rotate(-90deg);fill:#fbb03b}.trocket.revealed .st-2{transform:translate(37.5px,37.5px) scale(0.5,0.5) rotate(90deg);fill:#bdccd4}.trocket.revealed .mt{transform:translate(125px,25px) scale(-0.7142,0.7142) rotate(-135deg);fill:#e6e6e6}.trocket.revealed .sq{transform:translate(-37.5px,12.5px);fill:#4d4d4d}.trocket.revealed .rh{transform:translate(137px,137px) scale(1,-1) rotate(90deg);fill:#fbb03b}.tbird.revealed .bt-1{transform:translate(72.5px,-25px) rotate(45deg);fill:#d5abd8}.tbird.revealed .bt-2{transform:translate(72.5px,45.5px) scale(1,1) rotate(-135deg);fill:#d889db}.tbird.revealed .st-1{transform:translate(1.5px,45.5px) scale(0.5,0.5) rotate(-45deg);fill:#e6e6e6}.tbird.revealed .st-2{transform:translate(2px,-25px) scale(0.5,0.5) rotate(45deg);fill:#fbb03b}.tbird.revealed .mt{transform:translate(2px,-26px) scale(0.7142,0.7142) rotate(0deg);fill:#d29e98}.tbird.revealed .sq{transform:translate(36.5px,-25px) rotate(45deg);fill:#e6e6e6}.tbird.revealed .rh{transform:translate(107px,10px) scale(1,1) rotate(45deg);fill:#f2b1b7}.thome.revealed .bt-1{transform:translate(25px,66.5px) rotate(-90deg);fill:#d54d1d}.thome.revealed .bt-2{transform:translate(0px,116.5px) scale(1,1) rotate(-90deg);fill:#f8cc88}.thome.revealed .st-1{transform:translate(100px,66.5px) scale(0.5,0.5) rotate(90deg);fill:#fbe098}.thome.revealed .st-2{transform:translate(100px,116.5px) scale(0.5,0.5) rotate(180deg);fill:#f8cc88}.thome.revealed .mt{transform:translate(50px,66.5px) scale(-0.7142,0.7142) rotate(-45deg);fill:#fbe098}.thome.revealed .sq{transform:translate(25px,-75px) rotate(45deg);fill:#e6e6e6}.thome.revealed .rh{transform:translate(61px,137px) scale(1,-1) rotate(45deg);fill:#e9624e}.tlotus.revealed .bt-1{transform:translate(50px,85.5px) rotate(135deg);fill:#cddc39} .tlotus.revealed .bt-2{transform:translate(121.5px,14.5px) scale(1,1) rotate(45deg);fill:#8bc34a}.tlotus.revealed .st-1{transform:translate(50px,50px) scale(0.5,0.5) rotate(135deg);fill:#d889db}.tlotus.revealed .mt{transform:translate(86px,50px) scale(0.7142,0.7142) rotate(90deg);fill:#4caf50}.tlotus.revealed .st-2{transform:translate(50px,50px) scale(-0.5,0.5) rotate(135deg);fill:#f2b1b7}.tlotus.revealed .sq{transform:translate(-25px,-25px);fill:#de5e87}.tlotus.revealed .rh{transform:translate(121px,14.5px) rotate(45deg);fill:#009688}.tboat.revealed .bt-1{transform:translate(50px,12.5px) rotate(45deg);fill:#eee}.tboat.revealed .bt-2{transform:translate(50px,0) scale(1,1) rotate(0deg);fill:#ddd}.tboat.revealed .st-1{transform:translate(50px,100px) scale(0.5,0.5) rotate(-90deg);fill:#ededed}.tboat.revealed .mt{transform:translate(100px,100px) scale(0.7142,0.7142) rotate(90deg);fill:#795548}.tboat.revealed .st-2{transform:translate(85px,0px) scale(0.5,0.5) rotate(135deg);fill:#d27373}.tboat.revealed .sq{transform:translate(25px,25px);fill:#c7c4c4}.tboat.revealed .rh{transform:translate(64px,29.5px) rotate(45deg);fill:#593f35}.t{transition:opacity .75s .75s,background .5s;cursor:pointer;opacity:1}.hidden{cursor:auto;opacity:.5}</style>

</head>

<body>
	<!-- partial:index.partial.html -->
	<h1>Tangram Memory Game</h1>
	<p>Click cards and find matching tangrams</p>
	<div class="board" id="tangramgrid"></div>
	<svg style="display:none" viewBox="0 0 100 100">
	<defs>
	<polygon id="tangram1" points="0 0, 50 50, 0 100" />
	<polygon id="tangram2" points="50 50, 75 25, 100 50, 75 75" />
	<polygon id="tangram3" points="0 100, 25 75, 75 75, 50 100"/>
	</defs>
</svg>
	<!-- partial -->
	<script>
	  const classes = [
		'tcat',
		'tsquirrel',
		'theart',
		'trocket',
		'tbird',
		'thome',
		'tlotus',
		'tboat'
	];

	const tangram = ` + "`\n" +
		`<div class="t">
	  <svg viewbox="-50 -50 200 200" >
		<use class="bt-1" xlink:href="#tangram1" transform="translate(0,0)"/>
		<use class="bt-2 red" xlink:href="#tangram1" transform="translate(0,0) scale(-1,1) rotate(90 0 0)"/>
		<use class="st-1" xlink:href="#tangram1" transform="translate(25,75) scale(.5,.5) rotate(-90 0 0)"/>
		<use class="mt" xlink:href="#tangram1" transform="translate(100,49.5) scale(.7142,.7142) rotate(45 0 0)"/>
		<use class="st-2" xlink:href="#tangram1" transform="translate(100,50) scale(.5,.5) rotate(-180 0 0)"/>
		<use class="sq" xlink:href="#tangram2"/>
		<use class="rh" xlink:href="#tangram3"/>
	  </svg>
	</div>
	` + "`\n" +
		`class MemoryGame {
		constructor (selector) {
			this.selector = selector
			this.init()
		}
		init () {
			this.randomize()
			this.buildGrid()
			this.revealed = []
		}
		randomize () {
			this.classes = this.shuffle(classes.concat(classes))
		}
		shuffle (array) {
			array.sort( function(a, b) {return 0.5 - Math.random()} );
			return array;
		}
		buildGrid () {
			let html = ''
			for (let i = 0; i<this.classes.length; i++ ) {
				html += tangram;
			}
			this.selector.innerHTML = html;
			this.cards = this.selector.querySelectorAll('.t')
			this.classes.forEach ( (el, i) => {
				const card = this.cards[i]
				card.classList.add(el)
				card.setAttribute( 'data-class', el)
				this.addCardListeners(card)
			})

		}
		checkMatch () {
			return this.revealed.length === 2 && this.revealed[0].getAttribute('data-class') === this.revealed[1].getAttribute('data-class')
		}
		addCardListeners (el) {

			el.addEventListener( 'mouseenter', (e) => {

				if (el.classList.contains('revealed')){
					return
				}

				if ( this.revealed.length < 2 ) {
					return
				}

				Array.prototype.slice.call(this.revealed).forEach( (item) => {
					item.classList.remove('revealed')
				})
				this.revealed = []
			})

			el.addEventListener( 'click', (e) => {

				if (el.classList.contains('revealed')) {
					return
				}

				el.classList.add('revealed');
				this.revealed.push(el)
				if ( this.checkMatch() ) {
					Array.prototype.slice.call(this.revealed).forEach( (item) => {
					  item.classList.add('hidden')
					  this.revealed = []
				  })
				}

			}, false );

		}
	}

	let memory = new MemoryGame(document.getElementById("tangramgrid"));
    </script>

</body>

</html>`
)
// GameMemory 图形记忆游戏
func GameMemory(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.Status(http.StatusOK)
	render.WriteString(c.Writer, tplmemory, nil)
}
