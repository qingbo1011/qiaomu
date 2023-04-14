package qlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/qingbo1011/qiaomu/utils"
)

const (
	greenBg   = "\033[97;42m"
	whiteBg   = "\033[90;47m"
	yellowBg  = "\033[90;43m"
	redBg     = "\033[97;41m"
	blueBg    = "\033[97;44m"
	magentaBg = "\033[97;45m"
	cyanBg    = "\033[97;46m"
	green     = "\033[32m"
	white     = "\033[37m"
	yellow    = "\033[33m"
	red       = "\033[31m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
	reset     = "\033[0m"
)

// LoggerLevel 日志级别
type LoggerLevel int

const (
	LevelDebug LoggerLevel = iota // 0
	LevelInfo                     // 1
	LevelError                    // 2
)

// Level 获取日志级别
func (l LoggerLevel) Level() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	default:
		return ""
	}
}

type Fields map[string]any

// Logger 日志
type Logger struct {
	Formatter    LoggingFormatter
	Level        LoggerLevel
	Outs         []*LoggerWriter
	LoggerFields Fields
	logPath      string
	LogFileSize  int64
}

type LoggerWriter struct {
	Level LoggerLevel
	Out   io.Writer
}

// LoggingFormatter 日志格式化抽象接口
type LoggingFormatter interface {
	Format(param *LoggingFormatParam) string
}

type LoggingFormatParam struct {
	Level        LoggerLevel
	IsColor      bool
	LoggerFields Fields
	Msg          any
}

type LoggerFormatter struct {
	Level        LoggerLevel
	IsColor      bool
	LoggerFields Fields
}

// New 初始化Logger(没有任何如颜色等等的设置)
func New() *Logger {
	return &Logger{}
}

// Default 以默认配置初始化Logger(默认日志等级为LevelDebug，日志格式为文本格式，存储日志文件最大为100M,超过会新建日志文件)
func Default() *Logger {
	logger := New()
	logger.Level = LevelDebug
	w := &LoggerWriter{
		Level: LevelDebug,
		Out:   os.Stdout,
	}
	logger.Outs = append(logger.Outs, w)
	logger.Formatter = &TextFormatter{}
	logger.LogFileSize = 100 << 20
	return logger
}

// Debug Debug级别日志处理
func (l *Logger) Debug(msg any) {
	l.Print(LevelDebug, msg)
}

// Info Info级别日志处理
func (l *Logger) Info(msg any) {
	l.Print(LevelInfo, msg)
}

// Error Error级别日志处理
func (l *Logger) Error(msg any) {
	l.Print(LevelError, msg)
}

// Print 日志打印输出
func (l *Logger) Print(level LoggerLevel, msg any) {
	if l.Level > level { // 当前的级别大于输入级别，不打印对应的级别日志
		return
	}
	param := &LoggingFormatParam{
		Level:        level,
		LoggerFields: l.LoggerFields,
		Msg:          msg,
	}
	str := l.Formatter.Format(param)
	for _, out := range l.Outs {
		if out.Out == os.Stdout {
			param.IsColor = true
			str = l.Formatter.Format(param)
			fmt.Fprintln(out.Out, str)
		}
		if out.Level == -1 || level == out.Level {
			fmt.Fprintln(out.Out, str)
			l.CheckFileSize(out)
		}
	}
}

// WithFields 日志输出字段(在日志中打印一些字段信息，用于区分msg)
func (l *Logger) WithFields(fields Fields) *Logger {
	return &Logger{
		Formatter:    l.Formatter,
		Outs:         l.Outs,
		Level:        l.Level,
		LoggerFields: fields,
	}
}

// SetLogPath 设置日志存储路径并持久化存储日志
func (l *Logger) SetLogPath(logPath string) {
	now := time.Now()                         // 获取当前日期和时间
	formattedDate := now.Format("2006-01-02") // 将日期格式化为 "2006-01-02" 格式
	l.logPath = logPath
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: -1,
		Out:   FileWriter(path.Join(logPath, utils.ConcatenatedString([]string{formattedDate, "_all.log"}))),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: LevelDebug,
		Out:   FileWriter(path.Join(logPath, utils.ConcatenatedString([]string{formattedDate, "_debug.log"}))),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: LevelInfo,
		Out:   FileWriter(path.Join(logPath, utils.ConcatenatedString([]string{formattedDate, "_info.log"}))),
	})
	l.Outs = append(l.Outs, &LoggerWriter{
		Level: LevelError,
		Out:   FileWriter(path.Join(logPath, utils.ConcatenatedString([]string{formattedDate, "_error.log"}))),
	})
}

// CheckFileSize 日志文件大小，判断是否需要新建日志文件
func (l *Logger) CheckFileSize(w *LoggerWriter) {
	// 判断对应的文件大小
	logFile := w.Out.(*os.File)
	if logFile != nil {
		stat, err := logFile.Stat()
		if err != nil {
			log.Println(err)
			return
		}
		size := stat.Size()
		if l.LogFileSize <= 0 {
			l.LogFileSize = 100 << 20 // 100M
		}
		if size >= l.LogFileSize {
			_, name := path.Split(stat.Name())
			fileName := name[0:strings.Index(name, ".")]
			writer := FileWriter(path.Join(l.logPath, utils.JoinStrings(fileName, ".", time.Now().UnixMilli(), ".log")))
			w.Out = writer
		}
	}
}

func (f *LoggerFormatter) format(msg any) string {
	now := time.Now()
	if f.IsColor {
		// 要带颜色: error的颜色 为红色 info为绿色 debug为蓝色
		levelColor := f.LevelColor()
		msgColor := f.MsgColor()
		return fmt.Sprintf("%s [msgo] %s %s%v%s | level= %s %s %s | msg=%s %#v %s | fields=%v ",
			yellow, reset, blue, now.Format("2006/01/02 - 15:04:05"), reset,
			levelColor, f.Level.Level(), reset, msgColor, msg, reset, f.LoggerFields,
		)
	}
	return fmt.Sprintf("[msgo] %v | level=%s | msg=%#v | fields=%#v",
		now.Format("2006/01/02 - 15:04:05"),
		f.Level.Level(), msg, f.LoggerFields)
}

func (f *LoggerFormatter) LevelColor() string {
	switch f.Level {
	case LevelDebug:
		return blue
	case LevelInfo:
		return green
	case LevelError:
		return red
	default:
		return cyan
	}
}

func (f *LoggerFormatter) MsgColor() string {
	switch f.Level {
	case LevelError:
		return red
	default:
		return ""
	}
}

func FileWriter(name string) io.Writer {
	w, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	return w
}
