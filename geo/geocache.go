package geohash

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/xyzj/gopsu"
	"github.com/xyzj/gopsu/geo/sortedset"
	json "github.com/xyzj/gopsu/json"
)

// GeoCache geo数据缓存集
type GeoCache struct {
	cachename string
	sortedset *sortedset.SortedSet
	locker    *sync.RWMutex
}

// GeoPoint geo点
type GeoPoint struct {
	Name string  `json:"name"`
	Lng  float64 `json:"lng"`
	Lat  float64 `json:"lat"`
	Hash uint64
}

func (gp *GeoPoint) String() string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, gp.Hash)
	return enc.EncodeToString(buf)
}

type geoJSON struct {
	Points []*GeoPoint `json:"points"`
}

func getPoint(name string, hash uint64) *GeoPoint {
	gp := &GeoPoint{
		Name: name,
		Hash: hash,
	}
	gp.Lng, gp.Lat = Decode(hash)
	return gp
}

// GeoAdd 添加geo点
func (g *GeoCache) GeoAdd(points ...*GeoPoint) {
	if len(points) == 0 {
		return
	}
	g.locker.Lock()
	defer g.locker.Unlock()
	for _, point := range points {
		g.sortedset.Add(point.Name, Encode(point.Lng, point.Lat))
	}
}

// GeoRem 删除指定点
func (g *GeoCache) GeoRem(names ...string) {
	g.locker.Lock()
	defer g.locker.Unlock()
	for _, name := range names {
		g.sortedset.Remove(name)
	}
}

// GeoPos 返回指定名称的信息
func (g *GeoCache) GeoPos(names ...string) []*GeoPoint {
	if len(names) == 0 {
		return nil
	}
	g.locker.RLock()
	defer g.locker.RUnlock()
	gp := make([]*GeoPoint, len(names))
	idx := 0
	for _, name := range names {
		elem, exists := g.sortedset.Get(name)
		if !exists {
			continue
		}
		gp[idx] = getPoint(name, elem.Score)
		idx++
	}
	if idx == 0 {
		return nil
	}
	return gp[:idx]
}

// GeoDist 计算距离，单位米
func (g *GeoCache) GeoDist(name1, name2 string) (float64, error) {
	g.locker.RLock()
	defer g.locker.RUnlock()
	gp := g.GeoPos(name1, name2)
	if len(gp) != 2 {
		return 0, fmt.Errorf("point not found")
	}
	return Distance(gp[0].Lng, gp[0].Lat, gp[1].Lng, gp[1].Lat), nil
}

// GeoRadius 获取半径内的点
func (g *GeoCache) GeoRadius(longitude, latitude, radius float64) []*GeoPoint {
	g.locker.RLock()
	defer g.locker.RUnlock()
	areas := GetNeighbours(longitude, latitude, radius)
	gp := make([]*GeoPoint, 0)
	for _, area := range areas {
		lower := &sortedset.ScoreBorder{Value: area[0]}
		upper := &sortedset.ScoreBorder{Value: area[1]}
		elements := g.sortedset.RangeByScore(lower, upper, 0, -1, true)
		for _, elem := range elements {
			gp = append(gp, getPoint(elem.Member, elem.Score))
		}
	}
	return gp
}

// GeoRadiusByMember 获取指定成员半径内的点
func (g *GeoCache) GeoRadiusByMember(name string, radius float64) []*GeoPoint {
	gps := g.GeoPos(name)
	if gps == nil {
		return nil
	}
	g.locker.RLock()
	defer g.locker.RUnlock()
	areas := GetNeighbours(gps[0].Lng, gps[0].Lat, radius)
	gp := make([]*GeoPoint, 0)
	for _, area := range areas {
		lower := &sortedset.ScoreBorder{Value: area[0]}
		upper := &sortedset.ScoreBorder{Value: area[1]}
		elements := g.sortedset.RangeByScore(lower, upper, 0, -1, true)
		for _, elem := range elements {
			gp = append(gp, getPoint(elem.Member, elem.Score))
		}
	}
	return gp
}

// SaveToFile 保存到文件
func (g *GeoCache) SaveToFile() error {
	if g.cachename == "" {
		return fmt.Errorf("no file name was specified")
	}
	g.locker.RLock()
	defer g.locker.RUnlock()
	points := g.sortedset.Range(0, g.sortedset.Len(), false)
	gp := make([]*GeoPoint, len(points))
	for k, point := range points {
		gp[k] = getPoint(point.Member, point.Score)
	}
	geojson := &geoJSON{
		Points: gp,
	}
	b, err := json.Marshal(geojson)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(gopsu.JoinPathFromHere("_"+g.cachename), gopsu.CompressData(b, gopsu.ArchiveZlib), 0664)
}

// LoadFromFile 从文件读取缓存
func (g *GeoCache) LoadFromFile() error {
	if g.cachename == "" {
		return fmt.Errorf("no file name was specified")
	}
	b, err := ioutil.ReadFile(gopsu.JoinPathFromHere("_" + g.cachename))
	if err != nil {
		return nil
	}
	var geojson = &geoJSON{
		Points: make([]*GeoPoint, 0),
	}
	err = json.Unmarshal(gopsu.UncompressData(b, gopsu.ArchiveZlib), geojson)
	if err != nil {
		return err
	}
	g.GeoAdd(geojson.Points...)
	return nil
}

// Reset 重置geo缓存
func (g *GeoCache) Reset() {
	g.locker.Lock()
	defer g.locker.Unlock()
	g.sortedset = sortedset.Make()
}

// NewGeoCache 初始化一个新的geocache
func NewGeoCache(name string) *GeoCache {
	return &GeoCache{
		cachename: name,
		locker:    &sync.RWMutex{},
		sortedset: sortedset.Make(),
	}
}
