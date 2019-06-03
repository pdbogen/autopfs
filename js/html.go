package main

import (
	"encoding/json"
	"github.com/headzoo/surf/errors"
	"github.com/pdbogen/autopfs/types"
	"io"
	"io/ioutil"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"syscall/js"
)

var job types.Job
var sortCol, sortDir string

const (
	SortDirDesc  = "desc"
	SortDirAsc   = "asc"
	SortMarkDesc = " ▼"
	SortMarkAsc  = " ▲"
)

func UrlValueSet(key, value string) {
	loc := Location()
	values, err := url.ParseQuery(loc.RawQuery)
	if err != nil {
		println("could not parse RawQuery " + loc.RawQuery + ": " + err.Error())
	}
	if values == nil {
		values = url.Values{}
	}
	values.Set(key, value)
	loc.RawQuery = values.Encode()
	js.Global().Get("history").Call("pushState", nil, nil, loc.String())
}

type Response struct {
	Headers    map[string]string
	Status     int
	StatusText string
	Body       io.ReadCloser
}

func (r Response) String() string {
	var buf strings.Builder
	buf.WriteString("HTTP/1.1 ")
	buf.WriteString(strconv.Itoa(r.Status))
	buf.WriteRune(' ')
	buf.WriteString(r.StatusText)
	buf.WriteRune('\n')
	for k, v := range r.Headers {
		buf.WriteString(k)
		buf.WriteString(": ")
		buf.WriteString(v)
		buf.WriteRune('\n')
	}
	buf.WriteRune('\n')
	return buf.String()
}

type Stream struct {
	Stream js.Value
	reader js.Value
	ch     chan []byte
	buf    []byte
}

func (s *Stream) Read(to []byte) (int, error) {
	if (s.Stream == js.Value{}) {
		return 0, errors.New("underlying Stream is JS Undefined")
	}
	if s.reader == (js.Value{}) {
		s.reader = s.Stream.Call("getReader")
		s.ch = make(chan []byte)
	}

	if len(s.buf) > 0 {
		n := copy(to, s.buf)
		s.buf = s.buf[n:]
		return n, nil
	}

	s.reader.Call("read").Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			println("zero-length args in promise body")
			return nil
		}
		obj := args[0]
		done := obj.Get("done").Bool()

		if done {
			if s.ch != nil {
				close(s.ch)
				s.ch = nil
			}
			return nil
		}

		valueObj := obj.Get("value")
		value := make([]byte, valueObj.Length())
		for i := 0; i < valueObj.Length(); i++ {
			value[i] = byte(valueObj.Index(i).Int())
		}
		s.ch <- value
		return nil
	}))

	value, ok := <-s.ch
	if !ok {
		to = to[:0]
		return 0, io.EOF
	}

	n := copy(to, value)
	s.buf = []byte(value)[n:]
	return n, nil
}

func (s *Stream) Close() error {
	if s.reader != (js.Value{}) {
		s.reader.Call("cancel")
		s.reader = js.Value{}
		s.Stream = js.Value{}
		if s.ch != nil {
			close(s.ch)
			s.ch = nil
		}
	}

	return nil
}

var _ io.ReadCloser = (*Stream)(nil)

func fetch(url string) *Response {
	ch := make(chan js.Value)
	js.Global().Call("fetch", url).Call("then",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			ch <- args[0]
			close(ch)
			return nil
		}))
	resValue := <-ch
	res := &Response{
		Headers:    map[string]string{},
		Status:     resValue.Get("status").Int(),
		StatusText: resValue.Get("statusText").String(),
	}
	resValue.Get("headers").Call("forEach", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		res.Headers[args[1].String()] = args[0].String()
		return nil
	}))
	res.Body = &Stream{Stream: resValue.Get("body")}
	return res
}

func WireSort() {
	header := js.Global().Get("document").Call("getElementById", "jobTableHead")
	for _, column := range Columns {
		th := js.Global().Get("document").Call("createElement", "th")
		th.Set("className", "sorter")
		th.Set("id", "sort-"+column.Name)
		th.Set("innerHTML", strings.ReplaceAll(column.Name, " ", "&nbsp;"))
		th.Call("addEventListener", "click", HeaderClick(column.Name))
		header.Call("appendChild", th)
	}
}

func HeaderClick(name string) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if sortCol != name {
			sortDir = SortDirDesc // next checks will flip back to asc for a new column
			sortCol = name
		}

		if sortDir == SortDirAsc {
			sortDir = SortDirDesc
		} else {
			sortDir = SortDirAsc
		}

		UrlValueSet("sort", sortCol)
		UrlValueSet("dir", sortDir)

		Rerender()
		return nil
	})
}

func Sort(less func(i, j *types.Session) bool) {
	sort.Slice(job.Sessions, func(i, j int) bool {
		return less(job.Sessions[i], job.Sessions[j])
	})
}

func Html() {
	sortCol = Param("sort")
	sortDir = Param("dir")

	if filter := Param("filter"); filter != "" {
		if err := json.Unmarshal([]byte(filter), &Filters); err != nil {
			println("could not parse parameter filter: " + err.Error())
		}
	}

	WireSort()
	res := fetch("/json?id=" + Param("id"))
	defer res.Body.Close()

	jobsJson, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic("reading request body: " + err.Error())
	}

	if err := json.Unmarshal(jobsJson, &job); err != nil {
		panic("parsing request body: " + err.Error())
	}

	Rerender()
}

func SortData() {
	col := GetColumn(sortCol)
	if col == nil {
		println("SortData: no column " + sortCol)
		return
	}

	for _, col := range Columns {
		el := js.Global().Get("document").Call("getElementById", "sort-"+col.Name)
		innerhtml := col.Name
		if col.Name == sortCol {
			if sortDir == SortDirAsc {
				innerhtml += SortMarkAsc
			} else {
				innerhtml += SortMarkDesc
			}
		}
		el.Set("innerHTML", strings.ReplaceAll(innerhtml, " ", "&nbsp;"))
	}

	if sortDir == SortDirAsc {
		Sort(col.Less)
	} else {
		Sort(func(i, j *types.Session) bool {
			return col.Less(j, i)
		})
	}
}

func Rerender() {
	SortData()

	tbl := js.Global().Get("document").Call("getElementById", "jobTableBody")

	for {
		child := tbl.Get("firstChild")
		if child.Type() != js.TypeObject {
			break
		}
		tbl.Call("removeChild", child)
	}

sessions:
	for _, s := range job.Sessions {
		for _, filter := range Filters {
			for _, column := range Columns {
				if filter.Column != column.Name {
					continue
				}
				if column.Select == nil {
					println("have filter for column " + column.Name + " but column has no select fn")
					continue
				}
				if !column.Select(s, filter) {
					continue sessions
				}
			}
		}
		row := CreateElement("tr")
		for _, column := range Columns {
			row.Call("appendChild", column.Render(s))
		}
		tbl.Call("appendChild", row)
	}

	RerenderFilters(job.Sessions)
}

var Filters []Filter

func FillFilters() {
	println("FillFilters")
	form := js.Global().Get("document").Call("getElementById", "filtersForm")
	FillFiltersElem(form)

	active := js.Global().Get("document").Call("getElementById", "activeFilters")
	for {
		child := active.Get("firstChild")
		if child.Type() != js.TypeObject {
			break
		}
		active.Call("removeChild", child)
	}

	for i, filter := range Filters {
		span := CreateElementText(
			"span",
			filter.Column+" "+filter.Op+" "+strings.Join(filter.Values, ", "),
		)
		span.Set("className", "filter")
		closer := CreateElementText("span", "×")
		closer.Set("className", "filterRemove")
		closer.Set("index", i)
		closer.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			idx := this.Get("index").Int()
			Filters = append(Filters[0:idx], Filters[idx+1:]...)
			filtersJson, err := json.Marshal(Filters)
			if err != nil {
				println("could not marshal filters JSON: " + err.Error())
			} else {
				UrlValueSet("filter", string(filtersJson))
			}
			Rerender()
			return nil
		}))
		span.Call("appendChild", closer)
		active.Call("appendChild", span)
	}
}

func FillFiltersElem(parent js.Value) {
	println("in CalculateFiltersElem")
	if parent.Type() != js.TypeObject {
		return
	}

	child := parent.Get("firstChild")
	for child.Type() == js.TypeObject {
		switch tn := child.Get("tagName").String(); tn {
		case "undefined":
		case "BR":
		case "INPUT":
			switch child.Get("type").String() {
			case "text":
				col := child.Get("column").String()
				op := child.Get("op").String()
				for _, filter := range Filters {
					println("considering " + filter.Column + " for " + col)
					if filter.Column == col && filter.Op == op {
						println("filling " + col + " " + op)
						child.Set("value", filter.Values[0])
					}
				}
			}
		case "SELECT":
			col := child.Get("column").String()
			var values []string
			for _, filter := range Filters {
				if filter.Column == col {
					values = filter.Values
				}
			}
			opt := child.Get("firstChild")
			for {
				if opt.Type() != js.TypeObject {
					break
				}
				value := opt.Get("value").String()
				for _, selectedValue := range values {
					if value == selectedValue {
						opt.Set("selected", true)
						break
					}
				}

				opt = opt.Get("nextSibling")
			}
		case "DIV":
			FillFiltersElem(child)
		default:
			println("dunno what to do with a " + tn)
		}
		child = child.Get("nextSibling")
	}
}

func CalculateFilters() {
	println("CalculateFilters")
	form := js.Global().Get("document").Call("getElementById", "filtersForm")
	Filters = CalculateFiltersElem(form)

	filtersJson, err := json.Marshal(Filters)
	if err != nil {
		println("could not marshal filters JSON: " + err.Error())
	} else {
		UrlValueSet("filter", string(filtersJson))

	}
}

func CalculateFiltersElem(parent js.Value) []Filter {
	println("in CalculateFiltersElem")
	if parent.Type() != js.TypeObject {
		return nil
	}

	var ret []Filter

	child := parent.Get("firstChild")
	for child.Type() == js.TypeObject {
		switch tn := child.Get("tagName").String(); tn {
		case "undefined":
		case "BR":
		case "INPUT":
			switch child.Get("type").String() {
			case "text":
				val := child.Get("value").String()
				if val != "" {
					ret = append(ret, Filter{
						Column: child.Get("column").String(),
						Op:     child.Get("op").String(),
						Values: []string{val},
					})
				}
			}
		case "SELECT":
			var values []string
			opt := child.Get("firstChild")
			for {
				if opt.Type() != js.TypeObject {
					break
				}
				if opt.Get("selected").Bool() {
					values = append(values, opt.Get("value").String())
				}
				opt = opt.Get("nextSibling")
			}
			if len(values) > 0 {
				ret = append(ret, Filter{
					Column: child.Get("column").String(),
					Values: values,
				})
			}
		case "DIV":
			ret = append(ret, CalculateFiltersElem(child)...)
		default:
			println("dunno what to do with a " + tn)
		}
		child = child.Get("nextSibling")
	}
	return ret
}

func RerenderFilters(sessions []*types.Session) {
	filtDiv := js.Global().Get("document").Call("getElementById", "filters")
	for {
		child := filtDiv.Get("firstChild")
		if child.Type() != js.TypeObject {
			break
		}
		filtDiv.Call("removeChild", child)
	}

	form := CreateElement("form")
	form.Set("id", "filtersForm")
	form.Set("action", "#")
	form.Call("addEventListener", "submit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ev := args[0]
		ev.Call("preventDefault")
		CalculateFilters()
		Rerender()
		return false
	}))
	filtDiv.Call("appendChild", form)

	for _, col := range Columns {
		if col.FilterGadget != nil {
			filterSessions := sessions
			for _, filter := range Filters {
				if filter.Column == col.Name {
					break
				}
				filterSessions = filter.Apply(filterSessions)
			}
			label := CreateElementText("div", col.Name)
			label.Set("style", "font-weight: bold;")
			form.Call("appendChild", label)
			form.Call("appendChild", col.FilterGadget(filterSessions))
		}
	}

	FillFilters()

	sub := CreateElement("input")
	sub.Set("type", "submit")
	sub.Set("value", "Apply Filters")
	form.Call("appendChild", sub)
}
