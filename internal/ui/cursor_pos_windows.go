package ui

// besideCursor 計算寬 w、高 h 的視窗放在游標 (cx, cy) 旁邊的位置：優先放右下方，
// 放不下時改到左邊或上面，最後限制在工作區 work 之內。
func besideCursor(cx, cy, w, h int, work rect) (int, int) {
	left, top, right, bottom := int(work.Left), int(work.Top), int(work.Right), int(work.Bottom)
	x, y := cx+cursorOffsetPixels, cy+cursorOffsetPixels
	if x+w > right {
		x = cx - cursorOffsetPixels - w
	}
	if y+h > bottom {
		y = cy - cursorOffsetPixels - h
	}
	x = max(left, min(x, right-w))
	y = max(top, min(y, bottom-h))
	return x, y
}
