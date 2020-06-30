package gylog

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type LogFile struct {
	mu       sync.Mutex
	file     *os.File
	layout   string //time时间格式
	baseName string //log文件基本名称
	size     int64  //标记当前log文件写入的字节数，用于和maxSize对比
	seconds  int64  //清理多余log文件的时间间隔
	maxSize  int64  //log文件的最大尺寸，超过了就rotate
	maxCount int64  //可以保留的最大log文件数
}

func Open(baseName string, seconds, maxSize, maxCount int64) (io.WriteCloser, error) {
	return open(".20060102.150405", baseName, seconds, maxSize, maxCount)
}

func OpenDaily(baseName string) (io.WriteCloser, error) {
	return open(".20060102", baseName, 86400, 0, 0)
}

func open(layout, baseName string, seconds, maxSize, maxCount int64) (io.WriteCloser, error) {
	//此情况下禁用rotation，只打开log文件
	if seconds <= 0 && maxSize <= 0 && maxCount <= 0 {
		return openFile(baseName)
	}
	lf := &LogFile{
		layout:   layout,
		baseName: baseName,
		seconds:  seconds,
		maxSize:  maxSize,
		maxCount: maxCount,
	}
	if err := lf.rotate(); err != nil {
		ErrWarning(lf.Close())
		return nil, err
	}
	go lf.cycle()
	return lf, nil
}

func (lf *LogFile) Close() error {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	return lf.replace(nil)
}

func (lf *LogFile) Write(p []byte) (int, error) {
	lf.mu.Lock()
	defer lf.mu.Unlock()
	if lf.file == nil {
		return 0, os.ErrInvalid
	}
	n, err := lf.file.Write(p)
	lf.size += int64(n)
	if err != nil {
		return n, err
	}
	if lf.maxSize > 0 && lf.size > lf.maxSize {
		err = lf.rotate()
	}
	return n, err
}

func (lf *LogFile) replace(newFile *os.File) error {
	//先关闭之前打开的log文件，如果有的话
	if lf.file != nil {
		if err := lf.file.Close(); err != nil {
			return err
		}
	}
	//获取log文件size
	if newFile != nil {
		if stat, err := newFile.Stat(); err != nil {
			return err
		} else {
			lf.size = stat.Size()
		}
	} else {
		lf.size = 0
	}
	//保存此log文件
	lf.file = newFile
	return nil
}

func (lf *LogFile) rotate() error {
	name := lf.baseName + time.Now().Format(lf.layout)
	//已经打开同一个文件则返回
	if lf.file != nil && lf.file.Name() == name {
		return nil
	}
	file, err := openFile(name)
	if err == nil {
		err = lf.replace(file)
	}
	if err != nil {
		return err
	}
	//删除其他的非当前log文件
	go lf.purge()
	return nil
}

//循环调用rotate
func (lf *LogFile) cycle() {
	if lf.seconds <= 0 {
		return
	}
	ns := lf.seconds * 1e9
	for {
		<-time.After(time.Duration(ns))
		lf.mu.Lock()
		if lf.file == nil {
			lf.mu.Unlock()
			return
		} else {
			lf.mu.Unlock()
			ErrWarning(lf.rotate())
		}
	}
}

func (lf *LogFile) purge() {
	if lf.maxCount <= 0 {
		return
	}
	names, _ := filepath.Glob(lf.baseName + "*")
	i, n := 0, len(names)
	//移除文件名里面表示的时间有错误的文件
	for i < n {
		_, err := time.Parse(lf.layout, names[i][len(lf.baseName):])
		if err == nil {
			i++
		} else {
			names = append(names[:i], names[i+1:]...)
			n--
		}
	}
	n -= int(lf.maxCount)
	//移除当前log文件之外的其他log文件
	if n > 0 {
		//对log文件进行排序，时间早的在前，先删除时间早的log文件
		sort.Strings(names)
		for i = 0; i < n; i++ {
			if lf.file == nil || lf.file.Name() != names[i] {
				existWarning(os.Remove(names[i]))
			}
		}
	}
}

func openFile(name string) (*os.File, error) {
	return os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
}

func existWarning(err error) {
	if err != nil && !os.IsNotExist(err) {
		Warningln(err)
	}
}
