package walker

import (
	"fmt"
	"github.com/lxn/walk"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type MyMainWindow struct {
	*walk.MainWindow
	lb       *walk.ListBox
	te       *walk.TextEdit
	keywords *walk.LineEdit
	sbi      *walk.StatusBarItem
	outTE    *walk.TextEdit
	model    *Model
	query    *walk.PushButton
	page     int
	curtitle string
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

func (mw *MyMainWindow) aboutAction_Triggered() {
	walk.MsgBox(mw, "关于", "tuifer视频网站下载器\n作者：tuifer 完成时间：2021-3-25 日", walk.MsgBoxIconQuestion)
}

func (mw *MyMainWindow) search() {
	mw.GetList(-999)
}

func (mw *MyMainWindow) GetList(page int) {
	mw.page = page
	keywords := mw.keywords.Text()
	enkeywords := url.QueryEscape(keywords)
	defer func() {
		mw.lb.SetCurrentIndex(-1)
	}()
	go func() {
		//rf := mw.readFavorite()
		//m := &Model{items: rf.items}
		//mw.lb.SetModel(m)
		//mw.model = m
	}()
	fmt.Println(enkeywords)
	return
}

func (mw *MyMainWindow) lb_CurrentIndexChanged() {

	i := mw.lb.CurrentIndex()
	if i < 0 {
		return
	}
	//defer mw.lb.SetCurrentIndex(-1)
	item := &mw.model.items[i]
	s := strings.Split(item.value, "|")
	if len(s) == 2 {

		//记录当前页面的标题和链接
		mw.curtitle = item.name
	} else {
		mw.curtitle = ""
	}

	return
}

func (mw *MyMainWindow) lb_ItemActivated() {
	mw.curtitle = ""
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}
