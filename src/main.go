package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"fmt"
	"time"
	"sync"
	"math/rand"
	"io/ioutil"
	"encoding/json"
	"os"
	"image/png"
	"image"
	"image/color"
	"path/filepath"
)

const (
	FILE_INDEX = "res" + string(filepath.Separator) + "index.html"
	FILE_DATA  = "data" + string(filepath.Separator) + "data.json"
	FILE_PIC   = "res" + string(filepath.Separator) + "pic.png"

	CONVAS_WIDTH = 500
	CONVAS_HEIGH = 500

	AUTO_CTL_SEC = 6
)

var (
	inCh chan struct{}
	data *Data

	lock sync.RWMutex
)

const (
	DIR_UP    = iota
	DIR_DOWN
	DIR_LEFT
	DIR_RIGHT
	DIR_COUNT
)

type Data struct {
	CurPos   [2]int
	CurDir   int
	CurColor [3]uint8
	img      *image.NRGBA
}

func main() {
	load()
	inCh = make(chan struct{}, 64)
	go serve()

	e := gin.New()
	e.StaticFile("/pic.png", "res"+string(filepath.Separator)+"pic.png")
	e.GET("/", index)
	e.POST("/click", click)
	e.Run(":8801")
}

func serve() {
	tkrRefresh := time.NewTicker(time.Second * time.Duration(AUTO_CTL_SEC))
	tkrAuto := time.NewTicker(time.Second * time.Duration(AUTO_CTL_SEC*100))
	for {
		select {
		case <-inCh:
			manual()
		case <-tkrAuto.C:
			auto()
		case <-tkrRefresh.C:
			refresh()
			save()
		}
	}
}

//刷新
func refresh() {
	lock.Lock()
	defer lock.Unlock()
	//根据当前方向和颜色，步进一像素，如果超出边界了则方向相反一下
	newPos := data.CurPos
	switch data.CurDir {
	case DIR_UP:
		newPos[1]--
	case DIR_DOWN:
		newPos[1]++
	case DIR_LEFT:
		newPos[0]--
	case DIR_RIGHT:
		newPos[0]++
	}
	//check out of bounds
	if newPos[0] < 0 || newPos[0] >= CONVAS_WIDTH ||
		newPos[1] < 0 || newPos[1] >= CONVAS_HEIGH {
		switch data.CurDir {
		case DIR_UP:
			data.CurDir = DIR_DOWN
		case DIR_DOWN:
			data.CurDir = DIR_UP
		case DIR_LEFT:
			data.CurDir = DIR_RIGHT
		case DIR_RIGHT:
			data.CurDir = DIR_LEFT
		}
		return
	}
	data.CurPos = newPos
	data.img.Set(newPos[0], newPos[1], color.RGBA{
		R: data.CurColor[0],
		G: data.CurColor[1],
		B: data.CurColor[2],
		A: 255,
	})
}

//随机颜色和方向
func manual() {
	newDirIdx := rand.Intn(2)
	newColor := [3]uint8{}
	newColor[0] = uint8(rand.Intn(256))
	newColor[1] = uint8(rand.Intn(256))
	newColor[2] = uint8(rand.Intn(256))
	if newColor[0] == 255 &&
		newColor[1] == 255 &&
		newColor[2] == 255 {
		//change white to black
		newColor[0] = 0
		newColor[1] = 0
		newColor[2] = 0
	}
	lock.Lock()
	defer lock.Unlock()
	//update direction
	switch data.CurDir {
	case DIR_UP, DIR_DOWN:
		if newDirIdx == 0 {
			data.CurDir = DIR_LEFT
		} else {
			data.CurDir = DIR_RIGHT
		}
	case DIR_LEFT, DIR_RIGHT:
		if newDirIdx == 0 {
			data.CurDir = DIR_UP
		} else {
			data.CurDir = DIR_DOWN
		}
	}
	//update color
	data.CurColor = newColor
}

//随机方向
func auto() {
	newDirIdx := rand.Intn(2)
	lock.Lock()
	defer lock.Unlock()
	//update direction
	switch data.CurDir {
	case DIR_UP, DIR_DOWN:
		if newDirIdx == 0 {
			data.CurDir = DIR_LEFT
		} else {
			data.CurDir = DIR_RIGHT
		}
	case DIR_LEFT, DIR_RIGHT:
		if newDirIdx == 0 {
			data.CurDir = DIR_UP
		} else {
			data.CurDir = DIR_DOWN
		}
	}
}

//保存数据
func save() {
	lock.Lock()
	defer lock.Unlock()
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(FILE_DATA, b, 0666)
	if err != nil {
		panic(err)
	}
	outputImage(data.img)
}

//加载数据
func load() {
	lock.Lock()
	defer lock.Unlock()
	b, err := ioutil.ReadFile(FILE_DATA)
	if err != nil {
		panic(err)
	}
	data = &Data{}
	err = json.Unmarshal(b, data)
	if err != nil {
		panic(err)
	}
	f, err := os.Open(FILE_PIC)
	if err != nil || f == nil {
		//new
		data.img = image.NewNRGBA(image.Rect(0, 0, CONVAS_WIDTH-1, CONVAS_HEIGH-1))
		data.CurColor[0] = 255
		data.CurDir = DIR_RIGHT
		data.CurPos[0] = CONVAS_WIDTH / 2
		data.CurPos[1] = CONVAS_HEIGH / 2
	} else {
		defer f.Close()
		//decode
		img, err := png.Decode(f)
		if err != nil {
			panic(err)
		}
		data.img = img.(*image.NRGBA)
	}
}

func index(c *gin.Context) {
	//logInfo("index, remote addr[%v]", c.Request.RemoteAddr)

	c.File(FILE_INDEX)
}

func click(c *gin.Context) {
	logInfo("click, remote addr[%v]", c.Request.RemoteAddr)

	//push input cmd
	inCh <- struct{}{}

	c.Redirect(http.StatusMovedPermanently, "/")
}

func outputImage(img *image.NRGBA) {
	outFile, err := os.Create(FILE_PIC)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()
	png.Encode(outFile, img)
}

func logInfo(format string, a ...interface{}) {
	fmt.Printf("[%s] %v\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, a...))
}
