package util

import (
	"config"
	"errors"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Logger struct {
	Logfile   string
	Loghandle *os.File
	mu        sync.Mutex
}

var Applog = &Logger{Logfile: filepath.Dir(os.Args[0]) + "/../logs/" + strings.TrimSuffix(filepath.Base(os.Args[0]), path.Ext(os.Args[0])) + ".log"}

func init() {
	Applog.SetLogfile()
	var once sync.Once
	once.Do(func() { go Applog.logmonitor() })
}

func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// get path
func Dir(filename string) string {
	if runtime.GOOS == "windows" {
		index := strings.LastIndex(filename, "\\")
		return string([]byte(filename)[0:index])
	} else {
		return path.Dir(filename)
	}
}

func OpenFile(filename string, flag string) (*os.File, error) {

	var f *os.File
	var err error

	//create dir
	filedir := Dir(filename)
	if !CheckFileIsExist(filedir) {
		log.Printf("mkdir:%s", filedir)
		//os.MkdirAll(filedir, 0755)

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("md", filedir)
		} else {
			cmd = exec.Command("/bin/mkdir", "-p", filedir)
		}

		cmd.Run()
		//cmd.Wait()
	}

	//rewrite
	if flag == ">" {
		f, err = os.OpenFile(filename, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0755)
		f.Chown(os.Geteuid(), os.Getegid())
	} else if flag == ">>" {
		f, err = os.OpenFile(filename, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0755)
		f.Chown(os.Getuid(), os.Getgid())
	} else {
		err = errors.New("unknown flag:" + flag)
	}

	//return
	if err != nil {
		return nil, err
	} else {
		return f, nil
	}
}

func (a *Logger) GetLogger() *log.Logger {
	return log.New(a.Loghandle, "", log.Ldate|log.Ltime|log.Lshortfile)
}

func (a *Logger) SetLogfile() error {
	file, err := OpenFile(a.Logfile, ">>")
	if err != nil {
		Error(err)
	}
	a.Loghandle = file
	a.logformat()
	return err
}

func (a *Logger) logformat() {
	log.SetOutput(a.Loghandle)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func (a *Logger) logmonitor() {
	const d = 10 * time.Second

	t := time.NewTimer(d)
	for {
		select {
		case <-t.C:
		}
		if a.fileswitch() {
			a.mu.Lock()
			err := a.Loghandle.Close()
			if err == nil {
				e := os.Rename(a.Logfile, a.Logfile+"."+time.Now().Format("20060102150405"))
				if e == nil {
					a.SetLogfile()
				}
			}
			a.mu.Unlock()
		}
		t.Reset(d)
	}
}

func (a *Logger) fileswitch() bool {
	if filesize(a.Logfile) >= config.Logarchsize {
		return true
	}
	return false
}

func filesize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		return 0
	}

	return f.Size()
}

func Debug(v ...interface{}) {
	log.Println("[DEBUG]", v)
}

func Info(v ...interface{}) {
	log.Println("[INFO]", v)
}

func Warn(v ...interface{}) {
	log.Println("[WARN]", v)
}

func Error(v ...interface{}) {
	log.Println("[ERROR]", v)
}
