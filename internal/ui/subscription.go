package ui

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"myproxy.com/p/internal/database"
)

// SubscriptionPanel 管理订阅的显示和操作。
// 它使用双向数据绑定自动更新标签显示，支持添加、编辑和删除订阅。
type SubscriptionPanel struct {
	appState      *AppState
	tagContainer  fyne.CanvasObject // 标签容器（使用 HBox 以便动态更新）
	headerArea    fyne.CanvasObject // 头部区域（包含标签容器）
	subscriptions []*database.Subscription
}

// NewSubscriptionPanel 创建并初始化订阅管理面板。
// 该方法会创建标签容器、加载订阅列表，并设置数据绑定监听器。
// 参数：
//   - appState: 应用状态实例
//
// 返回：初始化后的订阅管理面板实例
func NewSubscriptionPanel(appState *AppState) *SubscriptionPanel {
	sp := &SubscriptionPanel{
		appState: appState,
	}

	// 创建标签容器（水平布局）
	sp.tagContainer = container.NewHBox()

	// 加载订阅列表
	sp.refreshSubscriptionList()

	// 监听绑定数据变化，自动更新标签显示
	appState.SubscriptionLabelsBinding.AddListener(binding.NewDataListener(func() {
		sp.updateTagsFromBinding()
	}))

	return sp
}

// Build 构建并返回订阅管理面板的 UI 组件。
// 返回：包含订阅标签和添加按钮的容器组件
func (sp *SubscriptionPanel) Build() fyne.CanvasObject {
	// 从绑定数据初始化标签显示
	sp.updateTagsFromBinding()

	// 按钮 - 添加图标
	addBtn := NewStyledButton("添加", theme.ContentAddIcon(), sp.onAddSubscription)
	updateBtn := NewStyledButton("更新订阅", theme.ViewRefreshIcon(), sp.onUpdateSubscription)

	// 订阅管理标题（使用标题样式）
	titleLabel := NewTitleLabel("订阅管理")

	// 订阅管理标题和标签组 - 优化布局和间距
	// 将标签容器放在一个带背景的容器中，使其更美观
	tagScroll := container.NewScroll(sp.tagContainer)
	tagScroll.SetMinSize(fyne.NewSize(300, 0)) // 设置最小宽度
	
	sp.headerArea = container.NewHBox(
		titleLabel,
		NewSpacer(SpacingLarge),
		tagScroll, // 使用滚动容器包装标签
		layout.NewSpacer(),
		addBtn,
		NewSpacer(SpacingSmall),
		updateBtn,
	)
	// 添加内边距
	sp.headerArea = container.NewPadded(sp.headerArea)

	return container.NewVBox(
		sp.headerArea,
		NewSeparator(),
	)
}

// updateTagsFromBinding 从绑定数据更新标签显示（使用双向绑定）
func (sp *SubscriptionPanel) updateTagsFromBinding() {
	// 从绑定数据获取标签列表
	labels, err := sp.appState.SubscriptionLabelsBinding.Get()
	if err != nil {
		// 如果获取失败，从数据库重新加载
		sp.refreshSubscriptionList()
		sp.appState.UpdateSubscriptionLabels()
		return
	}

	// 获取所有订阅（用于创建按钮的回调）
	sp.refreshSubscriptionList()

	// 创建新的标签按钮列表
	var tagButtons []fyne.CanvasObject

	// 为每个标签创建按钮
	for _, label := range labels {
		// 找到对应的订阅
		var sub *database.Subscription
		for _, s := range sp.subscriptions {
			if s.Label == label {
				sub = s
				break
			}
		}

		if sub != nil {
			// 创建标签按钮，点击时弹出编辑对话框
			// 使用带样式的按钮，标签按钮使用特殊样式
			tagBtn := widget.NewButton(label, func(s *database.Subscription) func() {
				return func() {
					sp.onEditSubscription(s)
				}
			}(sub))
			// 标签按钮使用中等重要性，使其更突出
			tagBtn.Importance = widget.MediumImportance
			// 优化标签按钮样式，使其更像标签/徽章
			// 添加图标使标签更美观
			tagBtn.SetIcon(theme.FolderIcon())
			tagButtons = append(tagButtons, tagBtn)
			// 添加小间距
			if len(tagButtons) > 1 {
				tagButtons = append(tagButtons, NewSpacer(SpacingSmall))
			}
		}
	}

	// 重新创建容器
	sp.tagContainer = container.NewHBox(tagButtons...)

	// 刷新 headerArea（如果已创建）
	// 注意：由于 Fyne 容器的不可变性，我们需要在主窗口级别刷新
	// 这里我们只是更新 tagContainer，主窗口会在需要时刷新
}

// onEditSubscription 编辑订阅（弹出对话框）
func (sp *SubscriptionPanel) onEditSubscription(sub *database.Subscription) {
	// 创建对话框内容 - 优化输入框样式
	urlEntry := widget.NewEntry()
	urlEntry.SetText(sub.URL)
	urlEntry.SetPlaceHolder("例如: https://example.com/subscribe")

	labelEntry := widget.NewEntry()
	labelEntry.SetText(sub.Label)
	labelEntry.SetPlaceHolder("例如: 我的订阅")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "订阅URL", Widget: urlEntry, HintText: "必填项"},
			{Text: "标签", Widget: labelEntry, HintText: "必填项"},
		},
	}

	// 创建对话框
	dialog.ShowForm("编辑订阅", "确定", "取消", form.Items, func(confirmed bool) {
		if !confirmed {
			return
		}

		url := urlEntry.Text
		label := labelEntry.Text

		// 验证必填项
		if url == "" {
			sp.showError("订阅URL不能为空")
			return
		}
		if label == "" {
			sp.showError("标签不能为空")
			return
		}

		// 如果URL改变，更新订阅
		if url != sub.URL {
			// 更新订阅
			err := sp.appState.SubscriptionManager.UpdateSubscription(url, label)
			if err != nil {
				sp.logAndShowError("订阅更新失败", err)
				return
			}
		} else if label != sub.Label {
			// 只更新标签
			_, err := database.AddOrUpdateSubscription(url, label)
			if err != nil {
				sp.logAndShowError("标签更新失败", err)
				return
			}
		}

		// 刷新订阅列表
		sp.refreshSubscriptionList()
		// 更新绑定数据，UI 会自动更新
		sp.appState.UpdateSubscriptionLabels()

		sp.appState.Window.SetTitle("订阅已更新")
	}, sp.appState.Window)
}

// onAddSubscription 添加订阅（弹出对话框）
func (sp *SubscriptionPanel) onAddSubscription() {
	// 创建对话框内容 - 优化输入框样式
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("例如: https://example.com/subscribe")

	labelEntry := widget.NewEntry()
	labelEntry.SetPlaceHolder("例如: 我的订阅")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "订阅URL", Widget: urlEntry, HintText: "必填项"},
			{Text: "标签", Widget: labelEntry, HintText: "必填项"},
		},
	}

	// 创建对话框
	dialog.ShowForm("添加订阅", "确定", "取消", form.Items, func(confirmed bool) {
		if !confirmed {
			return
		}

		url := urlEntry.Text
		label := labelEntry.Text

		// 验证必填项
		if url == "" {
			sp.showError("订阅URL不能为空")
			return
		}
		if label == "" {
			sp.showError("标签不能为空")
			return
		}

		// 获取订阅
		servers, err := sp.appState.SubscriptionManager.FetchSubscription(url, label)
		if err != nil {
			sp.logAndShowError("订阅获取失败", err)
			return
		}

		// 刷新订阅列表
		sp.refreshSubscriptionList()
		// 更新绑定数据，UI 会自动更新
		sp.appState.UpdateSubscriptionLabels()

		sp.appState.Window.SetTitle(fmt.Sprintf("订阅添加成功，共 %d 条服务器", len(servers)))
	}, sp.appState.Window)
}

// showError 显示错误对话框（统一错误处理）
func (sp *SubscriptionPanel) showError(message string) {
	if sp.appState != nil && sp.appState.Window != nil {
		dialog.ShowError(errors.New(message), sp.appState.Window)
	}
}

// logAndShowError 记录日志并显示错误对话框（统一错误处理）
func (sp *SubscriptionPanel) logAndShowError(message string, err error) {
	if sp.appState != nil && sp.appState.Logger != nil {
		sp.appState.Logger.Error("%s: %v", message, err)
	}
	if sp.appState != nil && sp.appState.Window != nil {
		dialog.ShowError(fmt.Errorf("%s: %w", message, err), sp.appState.Window)
	}
}

// refreshSubscriptionList 刷新订阅列表
func (sp *SubscriptionPanel) refreshSubscriptionList() {
	subscriptions, err := database.GetAllSubscriptions()
	if err != nil {
		// 如果数据库未初始化，使用空列表
		sp.subscriptions = []*database.Subscription{}
	} else {
		sp.subscriptions = subscriptions
	}
	// 注意：不再在这里刷新标签，而是通过绑定自动更新
}

// onUpdateSubscription 重新解析订阅URL并更新服务器列表
func (sp *SubscriptionPanel) onUpdateSubscription() {
	// 确保最新列表
	sp.refreshSubscriptionList()
	if len(sp.subscriptions) == 0 {
		sp.showError("暂无订阅可更新")
		return
	}

	// 选项列表
	options := make([]string, len(sp.subscriptions))
	for i, sub := range sp.subscriptions {
		options[i] = fmt.Sprintf("%s (%s)", sub.Label, sub.URL)
	}

	selectedIndex := 0
	selectWidget := widget.NewSelect(options, func(value string) {
		for i, opt := range options {
			if opt == value {
				selectedIndex = i
				break
			}
		}
	})
	selectWidget.SetSelected(options[0])

	formItems := []*widget.FormItem{
		{Text: "选择订阅", Widget: selectWidget},
	}

	dialog.ShowForm("更新订阅", "更新", "取消", formItems, func(confirmed bool) {
		if !confirmed {
			return
		}

		if selectedIndex < 0 || selectedIndex >= len(sp.subscriptions) {
			sp.showError("请选择要更新的订阅")
			return
		}

		sub := sp.subscriptions[selectedIndex]
		if err := sp.appState.SubscriptionManager.UpdateSubscription(sub.URL, sub.Label); err != nil {
			sp.logAndShowError("更新订阅失败", err)
			return
		}

		// 刷新订阅、服务器及状态显示
		sp.refreshSubscriptionList()
		sp.appState.UpdateSubscriptionLabels()
		// 从数据库重新同步服务器列表，确保UI与最新数据一致
		if sp.appState != nil {
			sp.appState.LoadServersFromDB()
		}
		if sp.appState != nil && sp.appState.MainWindow != nil {
			sp.appState.MainWindow.Refresh()
		}
		sp.appState.Window.SetTitle("订阅已更新")
	}, sp.appState.Window)
}
