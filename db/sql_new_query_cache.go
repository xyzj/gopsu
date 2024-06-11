package db

// QueryCache 查询缓存结果，返回QueryData结构
//
// cacheTag: 缓存标签
// startIdx: 起始行数
// rowCount: 查询的行数
func (d *Conn) QueryCache(cacheTag string, startRow, rowsCount int) *QueryData {
	if cacheTag == emptyCacheTag {
		return nil
	}
	if startRow < 1 {
		startRow = 1
	}
	if rowsCount < 0 {
		rowsCount = 0
	}
	query := &QueryData{
		CacheTag: cacheTag,
		Rows:     make([]*QueryDataRow, 0),
	}
	// 开始读取
	if src, ok := d.cfg.QueryCache.Load(cacheTag); ok {
		if msg := src; msg != nil {
			query.Total = msg.Total
			startRow = startRow - 1
			endRow := startRow + rowsCount
			if rowsCount == 0 || endRow > len(msg.Rows) {
				endRow = int(msg.Total)
			}
			if startRow >= int(msg.Total) {
				query.Total = 0
			} else {
				query.Total = msg.Total
				query.Rows = msg.Rows[startRow:endRow]
			}
		}
	} else {
		query.CacheTag = ""
	}
	return query
}

// QueryCacheMultirowPage 查询多行分页缓存结果，返回QueryData结构
//
// cacheTag: 缓存标签
// startIdx: 起始行数
// rowCount: 查询的行数
func (d *Conn) QueryCacheMultirowPage(cacheTag string, startRow, rowsCount, keyColumeID int) *QueryData {
	if cacheTag == emptyCacheTag {
		return nil
	}
	if keyColumeID == -1 {
		return d.QueryCache(cacheTag, startRow, rowsCount)
	}
	if startRow < 1 {
		startRow = 1
	}
	if rowsCount < 0 {
		rowsCount = 0
	}
	query := &QueryData{CacheTag: cacheTag}
	if src, ok := d.cfg.QueryCache.Load(cacheTag); ok {
		if msg := src; msg != nil {
			startRow = startRow - 1
			query.Total = msg.Total
			endRow := startRow + rowsCount
			if rowsCount == 0 {
				endRow = int(msg.Total)
			}
			if startRow >= int(msg.Total) {
				query.Total = 0
			} else {
				query.Total = msg.Total
				var rowIdx int
				var keyItem string
				for _, v := range msg.Rows {
					if keyItem == "" {
						keyItem = v.Cells[keyColumeID]
					}
					if keyItem != v.Cells[keyColumeID] {
						keyItem = v.Cells[keyColumeID]
						rowIdx++
					}
					if rowIdx >= startRow && rowIdx < endRow {
						query.Rows = append(query.Rows, v)
					}
				}
			}
		}
	}
	return query
}
