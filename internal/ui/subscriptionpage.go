package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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
	list          *widget.List
	content       fyne.CanvasObject
}

// NewSubscriptionPage 创建订阅管理页面
func NewSubscriptionPage(appState *AppState) *SubscriptionPage {
	sp := &SubscriptionPage{
		appState: appState,
	}
	sp.loadSubscriptions()
	return sp
}

// Build 构建订阅管理页面UI
func (sp *SubscriptionPage) Build() fyne.CanvasObject {
	// 1. 顶部导航栏 (简约风格)
	backBtn := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if sp.appState != nil && sp.appState.MainWindow != nil {
			sp.appState.MainWindow.ShowHomePage()
		}
	})
	backBtn.Importance = widget.LowImportance

	title := widget.NewLabelWithStyle("订阅管理", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	navBar := container.NewHBox(backBtn, title)

	// 2. 操作工具栏 (位于标题下方，紧凑设计)
	addBtn := widget.NewButtonWithIcon("新增订阅", theme.ContentAddIcon(), sp.showAddSubscriptionDialog)
	addBtn.Importance = widget.HighImportance

	batchUpdateBtn := widget.NewButtonWithIcon("全部更新", theme.ViewRefreshIcon(), sp.batchUpdateSubscriptions)
	batchUpdateBtn.Importance = widget.LowImportance

	actionToolbar := container.NewHBox(
		layout.NewSpacer(),
		batchUpdateBtn,
		addBtn,
	)

	// 组合头部区域
	headerStack := container.NewVBox(
		container.NewPadded(navBar),
		container.NewPadded(actionToolbar),
		canvas.NewLine(theme.SeparatorColor()),
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
	if sp.list != nil {
		sp.list.Refresh()
	}
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
					sp.appState.SubscriptionManager.UpdateSubscriptionByID(sub.ID)
				}
			}
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

	nameLabel  *widget.Label
	infoLabel  *widget.Label
	urlLabel   *widget.Label
	statusBar  *canvas.Rectangle

	updateBtn  *widget.Button
	editBtn    *widget.Button
	deleteBtn  *widget.Button
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
	card.deleteBtn.Importance = widget.DangerImportance

	card.renderObj = card.setupLayout()
	card.ExtendBaseWidget(card)
	return card
}

func (card *SubscriptionCard) setupLayout() fyne.CanvasObject {
	bg := canvas.NewRectangle(theme.InputBackgroundColor())
	bg.CornerRadius = 10

	// 文字信息排版
	textInfo := container.NewVBox(
		card.nameLabel,
		card.urlLabel,
		container.NewHBox(widget.NewIcon(theme.InfoIcon()), card.infoLabel),
	)

	// 右侧按钮组
	btnBox := container.NewHBox(
		card.updateBtn,
		card.editBtn,
		card.deleteBtn,
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
			card.page.appState.SubscriptionManager.UpdateSubscriptionByID(sub.ID)
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
				card.page.Refresh()
			}
		}, card.page.appState.Window)
	}
}

func (card *SubscriptionCard) showEditDialog() {
	urlEntry := widget.NewEntry()
	urlEntry.SetText(card.sub.URL)
	labelEntry := widget.NewEntry()
	labelEntry.SetText(card.sub.Label)

	items := []*widget.FormItem{
		{Text: "名称", Widget: labelEntry},
		{Text: "链接", Widget: urlEntry},
	}

	dialog.ShowForm("编辑订阅", "确认", "取消", items, func(ok bool) {
		if ok {
			// 基于唯一 ID 更新，即使 URL 相同也不会冲突
			database.UpdateSubscriptionByID(card.sub.ID, urlEntry.Text, labelEntry.Text)
			card.page.Refresh()
		}
	}, card.page.appState.Window)
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