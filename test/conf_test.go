package mxgo

import (
	"testing"

	mxgo ".."
)

func Test_LoadConfig(t *testing.T) {
	ans, ex := mxgo.LoadConfig("/home/xy/.xywork/golang/mxgo/test/test.conf")
	if ex != nil {
		t.Error(ex)
	} else {
		// ans.SetItem("xx", "1", "aasdkfhakhf")
		// ans.SetItem("aaaadf", "1", "aasdkfhakhf#ajsdfalfad#126531423#为鄂阿卡蒂芬哈佛的")
		// ans.SetItem("23", "1", "aasdkfhakhf")
		// ans.SetItem("asdf", "1", "aasdkfhakhf")
		// ans.SetItem("3214", "1", "aasdkfhakhf")
		ans.Save()
		println(ans.GetAll())
		// t.Log(fmt.Sprintf("load config %v", ans))
	}
}
