package coord

import (
	"math"
)

// WGS84ToCGCS2000 wgs84转CGCS2000
//
//	east：向东偏移差
//	meridian：中央子午经度
func WGS84ToCGCS2000(p *Point, east, meridian float64) *Point {
	L := p.Lng
	B := p.Lat

	a := 6378137.0           //椭球长半轴
	b := 6356752.3142451795  //椭球短半轴
	e := 0.081819190842621   //第一偏心率
	eC := 0.0820944379496957 //第二偏心率

	/*meridian := 0.0 //中央子午线经度
	n := 0    //带号
	if degree == 6 {
		//6度
		n = int((L + degree/2) / degree)
		meridian = degree*float64(n) - degree/2.0
	} else {
		//3度
		n = int(L / degree)
		meridian = degree * float64(n)
	}*/
	radB := B * math.Pi / 180.0 //纬度(弧度)
	//radL := L * math.Pi / 180.0          //经度(弧度)
	deltaL := (L - meridian) * math.Pi / 180.0 //经度差(弧度)
	N := a * a / b / math.Sqrt(1.0+eC*eC*math.Cos(radB)*math.Cos(radB))
	C1 := 1.0 + 3.0/4*e*e + 45.0/64*math.Pow(e, 4) + 175.0/256*math.Pow(e, 6) + 11025.0/16384*math.Pow(e, 8)
	C2 := 3.0/4*e*e + 15.0/16*math.Pow(e, 4) + 525.0/512*math.Pow(e, 6) + 2205.0/2048*math.Pow(e, 8)
	C3 := 15.0/64*math.Pow(e, 4) + 105.0/256*math.Pow(e, 6) + 2205.0/4096*math.Pow(e, 8)
	C4 := 35.0/512*math.Pow(e, 6) + 315.0/2048*math.Pow(e, 8)
	C5 := 315.0 / 131072 * math.Pow(e, 8)
	t := math.Tan(radB)
	eta := eC * math.Cos(radB)
	X := a * (1.0 - e*e) * (C1*radB - C2*math.Sin(2*radB)/2 + C3*math.Sin(4*radB)/4 - C4*math.Sin(6*radB)/6 + C5*math.Sin(8*radB))
	y2000 := X + N*math.Sin(radB)*math.Cos(radB)*math.Pow(deltaL, 2)*(1+math.Pow(deltaL*math.Cos(radB), 2)*(5-t*t+9*eta*eta+4*math.Pow(eta, 4))/12+math.Pow(deltaL*math.Cos(radB), 4)*(61-58*t*t+math.Pow(t, 4))/360)/2
	x2000 := N*deltaL*math.Cos(radB)*(1+math.Pow(deltaL*math.Cos(radB), 2)*(1-t*t+eta*eta)/6+math.Pow(deltaL*math.Cos(radB), 4)*(5-18*t*t+math.Pow(t, 4)-14*eta*eta-58*eta*eta*t*t)/120) + east
	return &Point{
		Lng: x2000,
		Lat: y2000,
	}
}

// WGS84ToCGCS2000v2 wgs84转CGCS2000另一个版本
//
//	east：向东偏移差
//	meridian：中央子午经度
func WGS84ToCGCS2000v2(p *Point, east, meridian float64) *Point {
	L := p.Lng
	B := p.Lat

	/*meridian := 0.0 //中央子午线经度
	n := 0    //带号
	if degree == 6 {
		//6度
		n = int((L + degree/2) / degree)
		meridian = degree*float64(n) - degree/2.0
	} else {
		//3度
		n = int(L / degree)
		meridian = degree * float64(n)
	}*/
	//NN曲率半径，测量学里面用N表示
	//M为子午线弧长，测量学里用大X表示
	//fai为底点纬度，由子午弧长反算公式得到，测量学里用Bf表示
	//R为底点所对的曲率半径，测量学里用Nf表示
	iPI := 0.0174532925199433 //3.1415926535898/180.0;
	a := 6378137.0
	f := 1 / 298.257222101 //CGCS2000坐标系参数
	//a=6378137.0; f=1/298.2572236; //wgs84坐标系参数
	longitude0 := meridian        //中央子午线 根据实际进行配置
	longitude0 = longitude0 * iPI //中央子午线转换为弧度
	longitude1 := L * iPI         //经度转换为弧度
	latitude1 := B * iPI          //纬度转换为弧度
	e2 := 2*f - f*f
	ee := e2 * (1.0 - e2)
	NN := a / math.Sqrt(1.0-e2*math.Sin(latitude1)*math.Sin(latitude1))
	T := math.Tan(latitude1) * math.Tan(latitude1)
	C := ee * math.Cos(latitude1) * math.Cos(latitude1)
	A := (longitude1 - longitude0) * math.Cos(latitude1)
	M := a * ((1-e2/4-3*e2*e2/64-5*e2*e2*e2/256)*latitude1 - (3*e2/8+3*e2*e2/32+45*e2*e2*e2/1024)*math.Sin(2*latitude1) + (15*e2*e2/256+45*e2*e2*e2/1024)*math.Sin(4*latitude1) - (35*e2*e2*e2/3072)*math.Sin(6*latitude1))
	xval := NN * (A + (1-T+C)*A*A*A/6 + (5-18*T+T*T+72*C-58*ee)*A*A*A*A*A/120)
	yval := M + NN*math.Tan(latitude1)*(A*A/2+(5-T+9*C+4*C*C)*A*A*A*A/24+(61-58*T+T*T+600*C-330*ee)*A*A*A*A*A*A/720)
	Y0 := 0.0
	xval = xval + east
	yval = yval + Y0
	return &Point{
		Lng: xval,
		Lat: yval,
	}
}

// CGCS2000ToWGS84v2 CGCS2000转wgs84
//
//	east：向东偏移差
//	meridian：中央子午经度
func CGCS2000ToWGS84v2(p *Point, east, meridian float64) *Point {
	y := p.Lng
	x := p.Lat
	L00 := math.Round(meridian/3.0) * 3.0
	meridian = L00 / 180 * 3.1415926 //中央带所在带的角度

	a := 6378137.0               //椭球长半轴
	efang := 0.0066943799901413  //square of e
	e2fang := 0.0067394967422764 //suqre of e2
	y = y - east

	//主曲率计算
	m0 := a * (1 - efang)
	m2 := 3.0 / 2.0 * efang * m0
	m4 := efang * m2 * 5.0 / 4.0
	m6 := efang * m4 * 7.0 / 6.0
	m8 := efang * m6 * 9.0 / 8.0

	//子午线曲率计算
	a0 := m0 + m2/2.0 + m4*3.0/8.0 + m6*5.0/16.0 + m8*35.0/128.0
	a2 := m2/2.0 + m4/2.0 + m6*15.0/32.0 + m8*7.0/16.0
	a4 := m4/8.0 + m6*3.0/16.0 + m8*7.0/32.0
	a6 := m6/32.0 + m8/16.0
	a8 := m8 / 128.0

	FBf := 0.0
	Bf0 := x / a0
	Bf1 := 0.0

	//计算Bf的值，直到满足条件
	for (Bf0 - Bf1) >= 0.0001 {
		Bf1 = Bf0
		FBf = -a2*math.Sin(2*Bf0)/2 + a4*math.Sin(4*Bf0)/4 - a6*math.Sin(6*Bf0)/6 + a8*math.Sin(8*Bf0)/8
		Bf0 = (x - FBf) / a0
	}
	Bf := Bf0
	//计算公式中参数
	Wf := math.Sqrt(1 - efang*math.Sin(Bf)*math.Sin(Bf))
	Nf := a / Wf
	Mf := a * (1 - efang) / math.Pow(Wf, 3)
	nffang := e2fang * math.Cos(Bf) * math.Cos(Bf)
	tf := math.Tan(Bf)
	B := Bf - tf*y*y/(2*Mf*Nf) + tf*(5+3*tf*tf+nffang-9*nffang*tf*tf)*math.Pow(y, 4)/(24*Mf*math.Pow(Nf, 3)) - tf*(61+90*tf*tf+45*math.Pow(tf, 4))*math.Pow(y, 6)/(720*Mf*math.Pow(Nf, 5))
	l := y/(Nf*math.Cos(Bf)) - (1+2*tf*tf+nffang)*math.Pow(y, 3)/(6*math.Pow(Nf, 3)*math.Cos(Bf)) + (5+28*tf*tf+24*math.Pow(tf, 4))*math.Pow(y, 5)/(120*math.Pow(Nf, 5)*math.Cos(Bf))
	L := l + meridian
	//转化成为十进制经纬度格式
	arrayB := rad2dms(B)
	arrayL := rad2dms(L)
	Bdec := dms2dec(arrayB)
	Ldec := dms2dec(arrayL)
	return &Point{
		Lat: Bdec,
		Lng: Ldec,
	}
}

// CGCS2000ToWGS84 CGCS2000转wgs84另一个版本
//
//	east：向东偏移差
//	meridian：中央子午经度
func CGCS2000ToWGS84(p *Point, east, meridian float64) *Point {
	Y := p.Lng
	X := p.Lat

	Y -= east
	iPI := 0.0174532925199433 //pi/180
	a := 6378137.0            //长半轴 m
	b := 6356752.31414        //短半轴 m
	//f := 1 / 298.257222101       //扁率 a-b/a
	e := 0.0818191910428         //第一偏心率 math.sqrt(5)
	ee := math.Sqrt(a*a-b*b) / b //第二偏心率
	bf := 0.0                    //底点纬度
	a0 := 1 + (3 * e * e / 4) + (45 * e * e * e * e / 64) + (175 * e * e * e * e * e * e / 256) + (11025 * e * e * e * e * e * e * e * e / 16384) + (43659 * e * e * e * e * e * e * e * e * e * e / 65536)
	b0 := X / (a * (1 - e*e) * a0)
	c1 := 3*e*e/8 + 3*e*e*e*e/16 + 213*e*e*e*e*e*e/2048 + 255*e*e*e*e*e*e*e*e/4096
	c2 := 21*e*e*e*e/256 + 21*e*e*e*e*e*e/256 + 533*e*e*e*e*e*e*e*e/8192
	c3 := 151*e*e*e*e*e*e*e*e/6144 + 151*e*e*e*e*e*e*e*e/4096
	c4 := 1097 * e * e * e * e * e * e * e * e / 131072
	bf = b0 + c1*math.Sin(2*b0) + c2*math.Sin(4*b0) + c3*math.Sin(6*b0) + c4*math.Sin(8*b0) // bf =b0+c1*sin2b0 + c2*sin4b0 + c3*sin6b0 +c4*sin8b0 +...
	tf := math.Tan(bf)
	n2 := ee * ee * math.Cos(bf) * math.Cos(bf) //第二偏心率平方成bf余弦平方
	c5 := a * a / b
	v := math.Sqrt(1 + ee*ee*math.Cos(bf)*math.Cos(bf))
	mf := c5 / (v * v * v) //子午圈半径
	nf := c5 / v           //卯酉圈半径
	//纬度计算
	lat := bf - (tf/(2*mf)*Y)*(Y/nf)*(1-1.0/12*(5+3*tf*tf+n2-9*n2*tf*tf)*(Y*Y/(nf*nf))+1.0/360*(61+90*tf*tf+45*tf*tf*tf*tf)*(Y*Y*Y*Y/(nf*nf*nf*nf)))
	//经度偏差
	lon := 1/(nf*math.Cos(bf))*Y - (1.0/(6*nf*nf*nf*math.Cos(bf)))*(1+2*tf*tf+n2)*Y*Y*Y + (1.0/(120*nf*nf*nf*nf*nf*math.Cos(bf)))*(5+28*tf*tf+24*tf*tf*tf*tf)*Y*Y*Y*Y*Y
	yval := lat / iPI
	xval := meridian + lon/iPI

	return &Point{
		Lat: yval,
		Lng: xval,
	}
}

// 将弧度转化为度分秒
func rad2dms(rad float64) [3]float64 {
	p := 180.0 / math.Pi * 3600
	dms := rad * p
	var a [3]float64
	a[0] = math.Floor(dms / 3600.0)
	a[1] = math.Floor((dms - a[0]*3600) / 60.0)
	a[2] = ((dms - a[0]*3600) - a[1]*60)
	return a
}

// 将度分秒转化为十进制坐标
func dms2dec(dms [3]float64) float64 {
	dec := 0.0
	dec = dms[0] + dms[1]/60.0 + dms[2]/3600.0
	return dec
}
