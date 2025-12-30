package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/database"
)

// SubscriptionPage 订阅管理页面
type SubscriptionPage struct {
	appState      *AppState
	subscriptions []*database.Subscription
	subscriptionsBinding binding.UntypedList // 订阅列表绑定
	list          *widget.List
	content       fyne.CanvasObject
}

// NewSubscriptionPage 创建订阅管理页面
func NewSubscriptionPage(appState *AppState) *SubscriptionPage {
	sp := &SubscriptionPage{
		appState:            appState,
		subscriptionsBinding: binding.NewUntypedList(),
	}
	sp.loadSubscriptions()
	
	// 监听绑定数据变化，自动刷新列表
	sp.subscriptionsBinding.AddListener(binding.NewDataListener(func() {
		if sp.list != nil {
			sp.list.Refresh()
		}
	}))
	
	return sp
}

// Build 构建订阅管理页面UI
func (sp *SubscriptionPage) Build() fyne.CanvasObject {
	// 1. 返回按钮
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if sp.appState != nil && sp.appState.MainWindow != nil {
			sp.appState.MainWindow.Back()
		}
	})
	backBtn.Importance = widget.LowImportance

	// 2. 操作工具栏 (替换标题栏位置)
	addBtn := widget.NewButtonWithIcon("新增订阅", theme.ContentAddIcon(), sp.showAddSubscriptionDialog)
	addBtn.Importance = widget.HighImportance

	batchUpdateBtn := widget.NewButtonWithIcon("全部更新", theme.ViewRefreshIcon(), sp.batchUpdateSubscriptions)
	batchUpdateBtn.Importance = widget.LowImportance

	// 合并返回按钮和操作工具栏到一行
	headerBar := container.NewHBox(
		backBtn,
		layout.NewSpacer(),
		addBtn,
		batchUpdateBtn,
	)

	// 组合头部区域
	headerStack := container.NewVBox(
		container.NewPadded(headerBar),
		canvas.NewLine(theme.Color(theme.ColorNameSeparator)),
	)

	// 3. 订阅列表 (支持滚动)
	sp.list = widget.NewList(
		sp.getSubscriptionCount,
		sp.createSubscriptionItem,
		sp.updateSubscriptionItem,
	)

	// 包装在滚动容器中并设置最小尺寸确保布局占满
	scrollList := container.NewScroll(sp.list)

	sp.content = container.NewBorder(
		headerStack,
		nil, nil, nil,
		container.NewPadded(scrollList),
	)

	return sp.content
}

func (sp *SubscriptionPage) loadSubscriptions() {
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		sp.subscriptions = []*database.Subscription{}
	} else {
		sp.subscriptions = subscriptions
	}
	
	// 更新绑定数据，触发 UI 自动刷新
	sp.updateSubscriptionsBinding()
}

// updateSubscriptionsBinding 更新订阅列表绑定数据
func (sp *SubscriptionPage) updateSubscriptionsBinding() {
	// 将订阅列表转换为 any 类型切片
	items := make([]any, len(sp.subscriptions))
	for i, sub := range sp.subscriptions {
		items[i] = sub
	}
	
	// 使用 Set 方法替换整个列表，这会触发绑定更新
	_ = sp.subscriptionsBinding.Set(items)
}

func (sp *SubscriptionPage) getSubscriptionCount() int {
	return len(sp.subscriptions)
}

func (sp *SubscriptionPage) createSubscriptionItem() fyne.CanvasObject {
	return NewSubscriptionCard(sp)
}

func (sp *SubscriptionPage) updateSubscriptionItem(id widget.ListItemID, obj fyne.CanvasObject) {
	if id < 0 || id >= len(sp.subscriptions) {
		return
	}
	card := obj.(*SubscriptionCard)
	card.Update(sp.subscriptions[id])
}

func (sp *SubscriptionPage) Refresh() {
	sp.loadSubscriptions()
	// 绑定数据更新后会自动触发列表刷新，无需手动调用
}

// showAddSubscriptionDialog 修复逻辑：支持添加重复URL作为新订阅
func (sp *SubscriptionPage) showAddSubscriptionDialog() {
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("https://...")
	labelEntry := widget.NewEntry()
	labelEntry.SetPlaceHolder("订阅名称")

	items := []*widget.FormItem{
		{Text: "名称", Widget: labelEntry},
		{Text: "链接", Widget: urlEntry},
	}

	d := dialog.NewForm("添加新订阅", "确定添加", "取消", items, func(ok bool) {
		if !ok || urlEntry.Text == "" {
			return
		}

		go func() {
			// 调用创建新订阅的逻辑（不根据URL去重）
			_, err := database.AddOrUpdateSubscription(urlEntry.Text, labelEntry.Text)
			if err != nil {
				fyne.Do(func() { dialog.ShowError(err, sp.appState.Window) })
				return
			}

			// 立即执行一次抓取
			if sp.appState.SubscriptionManager != nil {
				sp.appState.SubscriptionManager.FetchSubscription(urlEntry.Text, labelEntry.Text)
			}

			// 更新绑定数据，自动刷新 UI
			fyne.Do(func() { sp.Refresh() })
		}()
	}, sp.appState.Window)

	d.Resize(fyne.NewSize(420, 240))
	d.Show()
}

func (sp *SubscriptionPage) batchUpdateSubscriptions() {
	if len(sp.subscriptions) == 0 {
		return
	}
	dialog.ShowConfirm("批量更新", "确认更新所有订阅列表？", func(ok bool) {
		if !ok {
			return
		}
		go func() {
			for _, sub := range sp.subscriptions {
				if sp.appState.SubscriptionManager != nil {
					if err := sp.appState.SubscriptionManager.UpdateSubscriptionByID(sub.ID); err != nil {
						fyne.Do(func() {
							dialog.ShowError(fmt.Errorf("更新订阅失败: %w", err), sp.appState.Window)
						})
					}
				}
			}
			// 更新绑定数据，自动刷新 UI
			fyne.Do(func() { sp.Refresh() })
		}()
	}, sp.appState.Window)
}

// --- SubscriptionCard 内部组件 ---

type SubscriptionCard struct {
	widget.BaseWidget
	page      *SubscriptionPage
	sub       *database.Subscription
	renderObj fyne.CanvasObject

	nameLabel *widget.Label
	infoLabel *widget.Label
	urlLabel  *widget.Label
	statusBar *canvas.Rectangle

	updateBtn *widget.Button
	editBtn   *widget.Button
	deleteBtn *widget.Button
}

func NewSubscriptionCard(page *SubscriptionPage) *SubscriptionCard {
	card := &SubscriptionCard{page: page}

	card.nameLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	card.urlLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Italic: false})
	card.urlLabel.Truncation = fyne.TextTruncateEllipsis

	card.infoLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{})

	card.statusBar = canvas.NewRectangle(theme.PrimaryColor())
	card.statusBar.SetMinSize(fyne.NewSize(4, 0))

	// 微型化图标按钮
	card.updateBtn = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), nil)
	card.updateBtn.Importance = widget.LowImportance

	card.editBtn = widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), nil)
	card.editBtn.Importance = widget.LowImportance

	card.deleteBtn = widget.NewButtonWithIcon("", theme.DeleteIcon(), nil)
	card.deleteBtn.Importance = widget.DangerImportance // 红色警告背景，白色前景

	card.renderObj = card.setupLayout()
	card.ExtendBaseWidget(card)
	return card
}

func (card *SubscriptionCard) setupLayout() fyne.CanvasObject {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	bg.CornerRadius = 10

	// 文字信息排版
	textInfo := container.NewVBox(
		card.nameLabel,
		card.urlLabel,
		container.NewHBox(widget.NewIcon(theme.InfoIcon()), card.infoLabel),
	)

	// 右侧按钮组，水平排列，使用 Center 垂直居中避免占据整个容器高度
	btnBox := container.NewCenter(
		container.NewHBox(
			card.updateBtn,
			card.editBtn,
			card.deleteBtn,
		),
	)

	content := container.NewBorder(
		nil, nil,
		card.statusBar,
		btnBox,
		container.NewPadded(textInfo),
	)

	return container.NewStack(bg, content)
}

func (card *SubscriptionCard) Update(sub *database.Subscription) {
	card.sub = sub
	card.nameLabel.SetText(sub.Label)

	urlDisplay := sub.URL
	if len(urlDisplay) > 50 {
		urlDisplay = urlDisplay[:47] + "..."
	}
	card.urlLabel.SetText(urlDisplay)

	nodeCount, _ := database.GetServerCountBySubscriptionID(sub.ID)
	lastUpdate := "从未更新"
	if !sub.UpdatedAt.IsZero() {
		lastUpdate = card.formatTime(sub.UpdatedAt)
	}
	card.infoLabel.SetText(fmt.Sprintf("%d 节点 · 更新于 %s", nodeCount, lastUpdate))

	// 绑定事件 (基于 ID 操作)
	card.updateBtn.OnTapped = func() {
		card.updateBtn.Disable()
		go func() {
			if card.page.appState.SubscriptionManager != nil {
				if err := card.page.appState.SubscriptionManager.UpdateSubscriptionByID(sub.ID); err != nil {
					fyne.Do(func() {
						card.updateBtn.Enable()
						dialog.ShowError(fmt.Errorf("更新订阅失败: %w", err), card.page.appState.Window)
					})
					return
				}
			}
			// 更新绑定数据，自动刷新 UI
			fyne.Do(func() {
				card.updateBtn.Enable()
				card.page.Refresh()
			})
		}()
	}

	card.editBtn.OnTapped = card.showEditDialog

	card.deleteBtn.OnTapped = func() {
		msg := fmt.Sprintf("确定删除订阅 '%s' 吗？\n下属的 %d 个节点将被移除。", sub.Label, nodeCount)
		dialog.ShowConfirm("删除确认", msg, func(ok bool) {
			if ok {
				database.DeleteSubscription(sub.ID)
				// 更新绑定数据，自动刷新 UI
				card.page.Refresh()
			}
		}, card.page.appState.Window)
	}
}

func (card *SubscriptionCard) showEditDialog() {
	urlEntry := widget.NewEntry()
	urlEntry.SetText(card.sub.URL)
	urlEntry.SetPlaceHolder("https://...")
	labelEntry := widget.NewEntry()
	labelEntry.SetText(card.sub.Label)
	labelEntry.SetPlaceHolder("订阅名称")

	items := []*widget.FormItem{
		{Text: "名称", Widget: labelEntry},
		{Text: "链接", Widget: urlEntry},
	}

	d := dialog.NewForm("编辑订阅", "确认", "取消", items, func(ok bool) {
		if !ok || urlEntry.Text == "" {
			return
		}

		// 基于唯一 ID 更新，即使 URL 相同也不会冲突
		database.UpdateSubscriptionByID(card.sub.ID, urlEntry.Text, labelEntry.Text)
		// 更新绑定数据，自动刷新 UI
		card.page.Refresh()
	}, card.page.appState.Window)

	d.Resize(fyne.NewSize(420, 240))
	d.Show()
}

func (card *SubscriptionCard) formatTime(t time.Time) string {
	diff := time.Since(t)
	if diff < time.Minute {
		return "刚刚"
	} else if diff < time.Hour {
		return fmt.Sprintf("%d分钟前", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(diff.Hours()))
	}
	return t.Format("2006-01-02")
}

func (card *SubscriptionCard) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(card.renderObj)
}
