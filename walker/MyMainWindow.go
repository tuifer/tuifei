package walker

import (
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewWindow() *MyMainWindow {
	return &MyMainWindow{}
}
func Init() *MyMainWindow {
	mw := &MyMainWindow{}
	var items []LogEntry
	mw.LogModel = &LogModel{Loglist: items}
	if err := (MainWindow{
		Icon:     "img/search.ico",
		AssignTo: &mw.MainWindow,
		Title:    "烟火视频下载器-by tuifer",
		MenuItems: []MenuItem{
			Menu{
				Text: "&帮助",
				Items: []MenuItem{
					Separator{},
					Action{
						Text:        "爱奇艺",
						OnTriggered: mw.Iqiyi,
					},
					Action{
						Text:        "B站",
						OnTriggered: mw.Bilibili,
					},
					Action{
						Text:        "优酷",
						OnTriggered: mw.Youku,
					},
					Action{
						Text:        "抖音",
						OnTriggered: mw.Douyin,
					},
				},
			},
			Menu{
				Text: "&关于",
				Items: []MenuItem{
					Action{
						Text:        "作者",
						OnTriggered: mw.AboutAction_Triggered,
					},
				},
			},
			Menu{
				Text: "&退出",
				Items: []MenuItem{
					Separator{},
					Action{
						Text:        "退出",
						OnTriggered: func() { mw.Close() },
					},
				},
			},
		},
		MinSize: Size{1000, 600},
		Layout:  VBox{MarginsZero: true},

		Children: []Widget{
			Composite{
				MaxSize: Size{0, 50},
				Layout:  HBox{},
				Children: []Widget{
					Label{Text: "网址: "},
					LineEdit{
						AssignTo: &mw.InputUrl,
						Text:     "",
					},
					PushButton{
						AssignTo: &mw.Query,
						Text:     "解析",
					},
					PushButton{
						AssignTo: &mw.Down,
						Text:     "下载",
					},
					ProgressBar{
						AssignTo: &mw.PBar,
					},
				},
			},
			Composite{
				Layout: Grid{Columns: 2, Spacing: 10},
				Children: []Widget{
					Label{Text: "清晰度配置"},
					Label{Text: "执行日志"},
					ListBox{
						MaxSize:               Size{200, 0},
						AssignTo:              &mw.Lb,
						OnCurrentIndexChanged: mw.Lb_CurrentIndexChanged,
						OnItemActivated:       mw.Lb_ItemActivated,
					},
					ListBox{
						AssignTo:        &mw.Lb2,
						MultiSelection:  true,
						Model:           mw.LogModel,
						OnItemActivated: mw.Lb2_ItemActivated,
						ItemStyler: &Styler{
							Lb:                  &mw.Lb2,
							Model:               mw.LogModel,
							Dpi2StampSize:       make(map[int]walk.Size),
							WidthDPI2WsPerLine:  make(map[WidthDPI]int),
							TextWidthDPI2Height: make(map[TextWidthDPI]int),
						},
					},
				},
			},
		},
		StatusBarItems: []StatusBarItem{
			{
				AssignTo:    &mw.Sbi,
				Text:        "状态栏",
				ToolTipText: "no tooltip for me",
			},
		},
	}.Create()); err != nil {
		log.Fatal(err)
	}

	return mw

}

type MyMainWindow struct {
	*walk.MainWindow
	Lb       *walk.ListBox
	Lb2      *walk.ListBox
	te       *walk.TextEdit
	InputUrl *walk.LineEdit
	Sbi      *walk.StatusBarItem
	OutTE    *walk.TextEdit
	Model    *Model
	LogModel *LogModel
	Query    *walk.PushButton
	Down     *walk.PushButton
	PBar     *walk.ProgressBar
	Page     int
	CurKey   string
}

type Item struct {
	name  string
	value string
}

type Items struct {
	items []Item
}

type Model struct {
	walk.ListModelBase
	items []Item
}

func (m *Model) ItemCount() int {
	return len(m.items)
}

func (m *Model) Value(index int) interface{} {
	return m.items[index].name
}

func (mw *MyMainWindow) AboutAction_Triggered() {
	walk.MsgBox(mw, "关于", "烟火视频网站下载器\n作者：tuifer \n Email:tuifer@foxmail.com", walk.MsgBoxIconQuestion)
}

func (mw *MyMainWindow) Search() {
}
func (mw *MyMainWindow) SetList(s map[string]string) {
	data := make([]Item, 0, len(s))
	for i, ite := range s {
		data = append(data, Item{name: ite, value: i})
	}
	m := &Model{items: data}
	mw.Lb.SetModel(m)
	mw.Model = m
}
func (mw *MyMainWindow) LogAppend(log string) {
	trackLatest := mw.Lb2.ItemVisible(len(mw.LogModel.Loglist)-1) && len(mw.Lb2.SelectedIndexes()) <= 1

	mw.LogModel.Loglist = append(mw.LogModel.Loglist, LogEntry{time.Now(), log})
	index := len(mw.LogModel.Loglist) - 1
	mw.LogModel.PublishItemsInserted(index, index)

	if trackLatest {
		mw.Lb2.EnsureItemVisible(len(mw.LogModel.Loglist) - 1)
	}
	return
}
func (mw *MyMainWindow) EmptyStream() {
	mw.CurKey = ""
	return
}
func (mw *MyMainWindow) Lb_CurrentIndexChanged() {
	i := mw.Lb.CurrentIndex()
	if i < 0 {
		return
	}
	item := &mw.Model.items[i]
	mw.CurKey = item.value
	return
}
func (mw *MyMainWindow) Iqiyi() {

	walk.MsgBox(mw, "提示", "爱奇艺默认地址使用会员下载，只下载最高清晰度一种，\n免费资源可以多种清晰度选择下载，\n只需要将网址iqiyi.com前的i去掉即可", walk.MsgBoxIconInformation)
}
func (mw *MyMainWindow) Youku() {
	walk.MsgBox(mw, "提示", "优酷会员账号如果失效，请打开优酷网站登录上vip账号后，浏览器上按F12，\n在console面板输入console.log(document.cookie)复制输出内容，\n粘贴到tuifei.json修改youku的后面双引号中", walk.MsgBoxIconInformation)
}
func (mw *MyMainWindow) Bilibili() {
	walk.MsgBox(mw, "提示", "B站视频由于获取size会导致视频下载失效，所以b站视频下载无进度条", walk.MsgBoxIconInformation)

}
func (mw *MyMainWindow) Douyin() {
	walk.MsgBox(mw, "提示", "抖音视频已自动去掉水印", walk.MsgBoxIconInformation)

}
func (mw *MyMainWindow) Lb_ItemActivated() {
	value := mw.Model.items[mw.Lb.CurrentIndex()].value
	walk.MsgBox(mw, "提示", "已选择清晰度："+value, walk.MsgBoxIconInformation)
}

func (mw *MyMainWindow) Lb2_ItemActivated() {
	value := mw.LogModel.Loglist[mw.Lb2.CurrentIndex()].Message
	//walk.MsgBox(mw, "提示", "已选择清晰度："+value, walk.MsgBoxIconInformation)
	if err := walk.Clipboard().SetText(value); err != nil {
		log.Print("Copy: ", err)
	} else {
		walk.MsgBox(mw, "提示", "复制成功", walk.MsgBoxIconInformation)
	}
}
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

type LogModel struct {
	walk.ReflectListModelBase
	Loglist []LogEntry
}

func (m *LogModel) Items() interface{} {
	return m.Loglist
}

type LogEntry struct {
	Timestamp time.Time
	Message   string
}

type WidthDPI struct {
	width int // in native pixels
	dpi   int
}

type TextWidthDPI struct {
	text  string
	width int // in native pixels
	dpi   int
}

type Styler struct {
	Lb                  **walk.ListBox
	canvas              *walk.Canvas
	Model               *LogModel
	Font                *walk.Font
	Dpi2StampSize       map[int]walk.Size
	WidthDPI2WsPerLine  map[WidthDPI]int
	TextWidthDPI2Height map[TextWidthDPI]int // in native pixels
}

func (s *Styler) ItemHeightDependsOnWidth() bool {
	return true
}

func (s *Styler) DefaultItemHeight() int {
	dpi := (*s.Lb).DPI()
	marginV := walk.IntFrom96DPI(marginV96dpi, dpi)

	return s.StampSize().Height + marginV*2
}

const (
	marginH96dpi int = 6
	marginV96dpi int = 2
	lineW96dpi   int = 1
)

func (s *Styler) ItemHeight(index, width int) int {
	dpi := (*s.Lb).DPI()
	marginH := walk.IntFrom96DPI(marginH96dpi, dpi)
	marginV := walk.IntFrom96DPI(marginV96dpi, dpi)
	lineW := walk.IntFrom96DPI(lineW96dpi, dpi)

	msg := s.Model.Loglist[index].Message

	twd := TextWidthDPI{msg, width, dpi}

	if height, ok := s.TextWidthDPI2Height[twd]; ok {
		return height + marginV*2
	}

	canvas, err := s.Canvas()
	if err != nil {
		return 0
	}

	stampSize := s.StampSize()

	wd := WidthDPI{width, dpi}
	wsPerLine, ok := s.WidthDPI2WsPerLine[wd]
	if !ok {
		bounds, _, err := canvas.MeasureTextPixels("W", (*s.Lb).Font(), walk.Rectangle{Width: 9999999}, walk.TextCalcRect)
		if err != nil {
			return 0
		}
		wsPerLine = (width - marginH*4 - lineW - stampSize.Width) / bounds.Width
		s.WidthDPI2WsPerLine[wd] = wsPerLine
	}

	if len(msg) <= wsPerLine {
		s.TextWidthDPI2Height[twd] = stampSize.Height
		return stampSize.Height + marginV*2
	}

	bounds, _, err := canvas.MeasureTextPixels(msg, (*s.Lb).Font(), walk.Rectangle{Width: width - marginH*4 - lineW - stampSize.Width, Height: 255}, walk.TextEditControl|walk.TextWordbreak|walk.TextEndEllipsis)
	if err != nil {
		return 0
	}

	s.TextWidthDPI2Height[twd] = bounds.Height

	return bounds.Height + marginV*2
}

func (s *Styler) StyleItem(style *walk.ListItemStyle) {
	if canvas := style.Canvas(); canvas != nil {
		if style.Index()%2 == 1 && style.BackgroundColor == walk.Color(win.GetSysColor(win.COLOR_WINDOW)) {
			style.BackgroundColor = walk.Color(win.GetSysColor(win.COLOR_BTNFACE))
			if err := style.DrawBackground(); err != nil {
				return
			}
		}

		pen, err := walk.NewCosmeticPen(walk.PenSolid, style.LineColor)
		if err != nil {
			return
		}
		defer pen.Dispose()

		dpi := (*s.Lb).DPI()
		marginH := walk.IntFrom96DPI(marginH96dpi, dpi)
		marginV := walk.IntFrom96DPI(marginV96dpi, dpi)
		lineW := walk.IntFrom96DPI(lineW96dpi, dpi)

		b := style.BoundsPixels()
		b.X += marginH
		b.Y += marginV

		item := s.Model.Loglist[style.Index()]

		style.DrawText(item.Timestamp.Format(time.StampMilli), b, walk.TextEditControl|walk.TextWordbreak)

		stampSize := s.StampSize()

		x := b.X + stampSize.Width + marginH + lineW
		canvas.DrawLinePixels(pen, walk.Point{x, b.Y - marginV}, walk.Point{x, b.Y - marginV + b.Height})

		b.X += stampSize.Width + marginH*2 + lineW
		b.Width -= stampSize.Width + marginH*4 + lineW

		style.DrawText(item.Message, b, walk.TextEditControl|walk.TextWordbreak|walk.TextEndEllipsis)
	}
}

func (s *Styler) StampSize() walk.Size {
	dpi := (*s.Lb).DPI()

	stampSize, ok := s.Dpi2StampSize[dpi]
	if !ok {
		canvas, err := s.Canvas()
		if err != nil {
			return walk.Size{}
		}

		bounds, _, err := canvas.MeasureTextPixels("Jan _2 20:04:05.000", (*s.Lb).Font(), walk.Rectangle{Width: 9999999}, walk.TextCalcRect)
		if err != nil {
			return walk.Size{}
		}

		stampSize = bounds.Size()
		s.Dpi2StampSize[dpi] = stampSize
	}

	return stampSize
}

func (s *Styler) Canvas() (*walk.Canvas, error) {
	if s.canvas != nil {
		return s.canvas, nil
	}

	canvas, err := (*s.Lb).CreateCanvas()
	if err != nil {
		return nil, err
	}
	s.canvas = canvas
	(*s.Lb).AddDisposable(canvas)

	return canvas, nil
}
