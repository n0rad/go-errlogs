package erlog

import (
	"bytes"
	"fmt"
	"github.com/mgutz/ansi"
	"runtime"
	"sort"
	"strings"
	"time"
"github.com/n0rad/go-erlog/log"
	"io"
)

var pathSkip int = 0

var reset = ansi.ColorCode("reset")

var fileColorNormal = ansi.ColorCode("cyan+b")
var fileColorFail = ansi.ColorCode("cyan")

var timeColorNormal = ansi.ColorFunc("blue+b")
var timeColorFail = ansi.ColorFunc("blue")

var lvlColorError = ansi.ColorCode("red+b")
var lvlColorWarn = ansi.ColorCode("yellow+b")
var lvlColorInfo = ansi.ColorCode("green")
var lvlColorDebug = ansi.ColorCode("magenta")
var lvlColorTrace = ansi.ColorCode("blue")
var lvlColorPanic = ansi.ColorCode(":red+h")

type ErlogWriterAppender struct {
	out io.Writer
	level log.Level
}


func init() {
	_, file, _, _ := runtime.Caller(0)
	paths := strings.Split(file, "/")
	for i := 0; i < len(paths); i++ {
		if paths[i] == "github.com" {
			pathSkip = i + 2
			break
		}
	}
}

func NewErlogWriterAppender(writer io.Writer) (f *ErlogWriterAppender) {
	return &ErlogWriterAppender{
		out: writer,
	}
}

func (f *ErlogWriterAppender) GetLevel() log.Level {
	return f.level
}

func (f *ErlogWriterAppender) SetLevel(level log.Level) {
 	f.level = level
}

func (f *ErlogWriterAppender) Fire(event *LogEvent) {
	keys := f.prepareKeys(event)
	time := time.Now().Format("15:04:05")
	level := f.textLevel(event.Level)

	//	isColored := isTerminal && (runtime.GOOS != "windows")
//	paths := strings.SplitN(event.File, "/", pathSkip+1)

	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s %s%-5s%s %s%30s:%-3d%s %s%-44s%s",
		f.timeColor(event.Level)(time),
		f.levelColor(event.Level),
		level,
		reset,
		f.fileColor(event.Level),
//		f.reduceFilePath(paths[pathSkip], 30),
		event.File,
		event.Line,
		reset,
		f.textColor(event.Level),
		event.Message,
		reset)
	for _, k := range keys {
		v := event.Entry.Fields[k]
		fmt.Fprintf(b, " %s%s%s=%+v", lvlColorInfo, k, reset, v)
	}
	b.WriteByte('\n')

	f.out.Write(b.Bytes())
}

func (f *ErlogWriterAppender) reduceFilePath(path string, max int) string {
	if len(path) <= max {
		return path
	}

	split := strings.Split(path, "/")
	splitlen := len(split)
	reducedSize := len(path)
	var buffer bytes.Buffer
	for i, e := range split {
		if reducedSize > max && i+1 < splitlen {
			buffer.WriteByte(e[0])
			reducedSize -= len(e) - 1
		} else {
			buffer.WriteString(e)
		}
		if i+1 < splitlen {
			buffer.WriteByte('/')
		}
	}
	return buffer.String()
}

func (f *ErlogWriterAppender) prepareKeys(event *LogEvent) []string {
	var keys []string = make([]string, 0, len(event.Entry.Fields))
	for k := range event.Entry.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (f *ErlogWriterAppender) textLevel(level log.Level) string {
	levelText := strings.ToUpper(level.String())
	switch level {
	case log.INFO:
	case log.WARN:
		levelText = levelText[0:4]
	default:
		levelText = levelText[0:5]
	}
	return levelText
}

func (f *ErlogWriterAppender) fileColor(level log.Level) string {
	switch level {
	case log.DEBUG, log.INFO, log.TRACE:
		return fileColorFail
	default:
		return fileColorNormal
	}
}

func (f *ErlogWriterAppender) textColor(level log.Level) string {
	switch level {
	case log.WARN:
		return lvlColorWarn
	case log.ERROR, log.FATAL, log.PANIC:
		return lvlColorError
	default:
		return ""
	}
}

func (f *ErlogWriterAppender) timeColor(level log.Level) func(string) string {
	switch level {
	case log.DEBUG, log.INFO:
		return timeColorFail
	default:
		return timeColorNormal
	}
}

func (f *ErlogWriterAppender) levelColor(level log.Level) string {
	switch level {
	case log.TRACE:
		return lvlColorTrace
	case log.DEBUG:
		return lvlColorDebug
	case log.WARN:
		return lvlColorWarn
	case log.ERROR:
		return lvlColorError
	case log.FATAL, log.PANIC:
		return lvlColorPanic
	default:
		return lvlColorInfo
	}
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func (f *ErlogWriterAppender) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	b.WriteString(key)
	b.WriteByte('=')

	switch value := value.(type) {
	case string:
		if needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case error:
		errmsg := value.Error()
		if needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	default:
		fmt.Fprint(b, value)
	}

	b.WriteByte(' ')
}
