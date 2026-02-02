package ui

import (
	"fyne.io/fyne/v2"
)

type CustomBorderLayout struct {
	topHeightPercent    float64
	bottomHeightPercent float64
}

func NewCustomBorderLayout(topPercent, bottomPercent float64) *CustomBorderLayout {
	return &CustomBorderLayout{
		topHeightPercent:    topPercent,
		bottomHeightPercent: bottomPercent,
	}
}

func (c *CustomBorderLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	defer func() {
		if r := recover(); r != nil {
			_ = r
		}
	}()

	if len(objects) != 5 {
		return // Border 布局需要 5 个对象：top, bottom, left, right, center
	}

	// 安全地获取对象，防止 nil 指针
	top := objects[0]
	bottom := objects[1]
	left := objects[2]
	right := objects[3]
	center := objects[4]

	// 如果中心组件为 nil，直接返回，避免后续操作导致崩溃
	if center == nil {
		return
	}

	// 确保尺寸有效
	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	// 计算顶部和底部的高度（严格按照百分比）
	topHeight := size.Height * float32(c.topHeightPercent)
	bottomHeight := size.Height * float32(c.bottomHeightPercent)

	// 确保顶部最小高度
	if top != nil {
		// 安全地检查可见性和获取最小尺寸
		if top.Visible() {
			minTopSize := top.MinSize()
			if minTopSize.Height > 0 {
				minTopHeight := minTopSize.Height
				if topHeight < minTopHeight && minTopHeight < size.Height*0.3 {
					topHeight = minTopHeight
				}
			}
		}
	}

	// 底部区域必须严格按照百分比显示，确保始终可见
	// 优先使用百分比计算的高度，但确保至少有一个最小可见高度
	if bottom != nil {
		// 强制显示底部区域
		if !bottom.Visible() {
			bottom.Show()
		}

		// 计算出的百分比高度
		calculatedBottomHeight := size.Height * float32(c.bottomHeightPercent)

		// 使用计算出的百分比高度，但确保至少50像素（确保内容可见）
		bottomHeight = calculatedBottomHeight
		if bottomHeight < 50 {
			bottomHeight = 50
		}

		// 确保不会超出窗口范围
		maxBottomHeight := size.Height - topHeight - 10 // 留10像素缓冲
		if bottomHeight > maxBottomHeight && maxBottomHeight > 0 {
			bottomHeight = maxBottomHeight
		}

		// 再次确保最小高度
		if bottomHeight < 50 {
			bottomHeight = 50
		}
	}

	// 计算左侧和右侧的宽度
	leftWidth := float32(0)
	rightWidth := float32(0)
	if left != nil && left.Visible() {
		leftWidth = left.MinSize().Width
	}
	if right != nil && right.Visible() {
		rightWidth = right.MinSize().Width
	}

	// 计算中间区域的大小
	centerWidth := size.Width - leftWidth - rightWidth
	centerHeight := size.Height - topHeight - bottomHeight

	// 确保中间区域高度至少为0（防止负数）
	if centerHeight < 0 {
		centerHeight = 0
	}

	// 布局顶部
	if top != nil {
		// 安全地检查和操作
		if top.Visible() {
			top.Resize(fyne.NewSize(size.Width, topHeight))
			top.Move(fyne.NewPos(0, 0))
		}
	}

	// 布局底部 - 确保始终可见
	if bottom != nil {
		// 强制显示底部区域，即使不可见也显示
		if !bottom.Visible() {
			bottom.Show()
		}

		// 确保底部高度至少为计算出的百分比高度
		calculatedBottomHeight := size.Height * float32(c.bottomHeightPercent)
		if bottomHeight < calculatedBottomHeight {
			bottomHeight = calculatedBottomHeight
		}
		// 确保最小可见高度（至少50像素，确保内容可见）
		if bottomHeight < 50 {
			bottomHeight = 50
		}
		// 确保不会超出窗口范围
		if bottomHeight > size.Height-topHeight {
			bottomHeight = size.Height - topHeight - 10 // 留10像素缓冲
		}
		// 确保底部高度至少为50像素
		if bottomHeight < 50 {
			bottomHeight = 50
		}
		bottom.Resize(fyne.NewSize(size.Width, bottomHeight))
		bottom.Move(fyne.NewPos(0, size.Height-bottomHeight))
	}

	// 布局左侧
	if left != nil && left.Visible() {
		left.Resize(fyne.NewSize(leftWidth, centerHeight))
		left.Move(fyne.NewPos(0, topHeight))
	}

	// 布局右侧
	if right != nil && right.Visible() {
		right.Resize(fyne.NewSize(rightWidth, centerHeight))
		right.Move(fyne.NewPos(size.Width-rightWidth, topHeight))
	}

	// 布局中间 - 必须存在且有效
	// 注意：center 已经在函数开头检查过，这里再次确认
	if center != nil {
		// 确保尺寸有效（至少要有最小尺寸）
		if centerWidth < 0 {
			centerWidth = 100 // 最小宽度
		}
		if centerHeight < 0 {
			centerHeight = 100 // 最小高度
		}

		// 确保有合理的尺寸
		if centerWidth > 0 && centerHeight > 0 {
			// 使用 defer recover 来捕获可能的 panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						// 如果刷新时出现 panic，记录但不崩溃
						// 这在窗口移动时可能会发生
					}
				}()

				// 安全地检查和操作
				if center.Visible() {
					center.Resize(fyne.NewSize(centerWidth, centerHeight))
					center.Move(fyne.NewPos(leftWidth, topHeight))
				} else {
					// 如果不可见，尝试显示它
					center.Show()
					center.Resize(fyne.NewSize(centerWidth, centerHeight))
					center.Move(fyne.NewPos(leftWidth, topHeight))
				}
			}()
		}
	}
}

// MinSize 计算最小尺寸
func (c *CustomBorderLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 5 {
		return fyne.NewSize(0, 0)
	}

	top := objects[0]
	bottom := objects[1]
	left := objects[2]
	right := objects[3]
	center := objects[4]

	// 如果中心组件为 nil，返回最小尺寸
	if center == nil {
		return fyne.NewSize(100, 100) // 返回一个合理的最小尺寸
	}

	minWidth := float32(0)
	minHeight := float32(0)

	// 顶部最小高度
	if top != nil && top.Visible() {
		minTopSize := top.MinSize()
		minHeight += minTopSize.Height
		minWidth = fyne.Max(minWidth, minTopSize.Width)
	}

	// 底部最小高度
	if bottom != nil && bottom.Visible() {
		minBottomSize := bottom.MinSize()
		minHeight += minBottomSize.Height
		minWidth = fyne.Max(minWidth, minBottomSize.Width)
	}

	// 左侧和右侧宽度
	leftRightWidth := float32(0)
	if left != nil && left.Visible() {
		leftRightWidth += left.MinSize().Width
	}
	if right != nil && right.Visible() {
		leftRightWidth += right.MinSize().Width
	}

	// 中间区域最小尺寸
	centerMinSize := fyne.NewSize(0, 0)
	if center != nil && center.Visible() {
		centerMinSize = center.MinSize()
	}

	// 总最小宽度和高度
	totalWidth := fyne.Max(minWidth, centerMinSize.Width+leftRightWidth)
	totalHeight := minHeight + centerMinSize.Height

	return fyne.NewSize(totalWidth, totalHeight)
}
