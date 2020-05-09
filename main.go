package main

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	_ "image/png"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/kbinani/screenshot"
)

type imageAlert struct {
	file  string
	img   image.Image
	state bool
}

func main() {
	f, err := os.Open("sounds/tindeck_1.mp3")
	if err != nil {
		panic(err)
	}
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		panic(err)
	}
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	defer streamer.Close()
	imgs, err := loadImages()
	if err != nil {
		panic(err)
	}
	scanTicker := time.NewTicker(30 * time.Millisecond)
	for {
		select {
		case <-scanTicker.C:
			search(imgs, streamer)
		}
	}
}

func search(imgAlerts []imageAlert, streamer beep.StreamSeekCloser) {
	n := screenshot.NumActiveDisplays()
	for i, searchImgAlert := range imgAlerts {
		alertRef := &imgAlerts[i]
		found := false
		for displayN := 0; displayN < n; displayN++ {
			bounds := screenshot.GetDisplayBounds(displayN)

			img, err := screenshot.CaptureRect(bounds)
			if err != nil {
				panic(err)
			}
			found = found || findMatch(img, searchImgAlert.img, img.Bounds())
			if found {
				break
			}
		}
		if found && !alertRef.state {
			fmt.Println("Notifying found " + alertRef.file)
			speaker.Play(streamer)
			streamer.Seek(0)
		} else if !found && alertRef.state {
			fmt.Println("Notifying not found " + alertRef.file)
			speaker.Play(streamer)
			streamer.Seek(0)
		}
		alertRef.state = found
	}
}

func loadImages() ([]imageAlert, error) {
	images := make([]imageAlert, 0)
	files, err := loadFiles()
	if err != nil {
		return images, err
	}
	for _, file := range files {
		fmt.Printf("Loading %s\n", file)
		infile, err := os.Open(file)
		if err != nil {
			return images, err
		}
		img, _, err := image.Decode(infile)
		if err != nil {
			return images, err
		}
		ia := imageAlert{img: img, state: false, file: file}
		images = append(images, ia)
		infile.Close()
	}
	return images, err
}

func loadFiles() ([]string, error) {
	var files []string

	root := "resources"
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return err
	})
	return files, err
}

func findMatch(container image.Image, find image.Image, withinBounds image.Rectangle) bool {
	var margin uint32
	margin = 0
	found := false
	endX := withinBounds.Max.X
	endY := withinBounds.Max.Y
	//Iterate pixels within container image and specified bounds
	for startY := withinBounds.Min.Y; startY <= endY; startY++ {
		for startX := withinBounds.Min.X; startX <= endX; startX++ {
			//Check if image matches

			found = true
			yCount := 0
			for searchImageY := find.Bounds().Min.Y; searchImageY <= find.Bounds().Max.Y; searchImageY++ {
				xCount := 0
				for searchImageX := find.Bounds().Min.X; searchImageX <= find.Bounds().Max.X; searchImageX++ {
					sColor := find.At(searchImageX, searchImageY)
					tColor := container.At(startX+xCount, startY+yCount)
					xCount++
					sR, sG, sB, sA := sColor.RGBA()
					if sA == 0 {
						continue
					}
					tR, tG, tB, _ := tColor.RGBA()
					same := (sR >= tR-margin && sR <= tR+margin) && (sG >= tG-margin && sG <= tG+margin) && (sB >= tB-margin && sB <= tB+margin)
					if !same {
						found = false
						break
					}
				}
				if !found {
					break
				}
				yCount++
			}
			if found {
				break
			}
		}
		if found {
			break
		}
	}
	return found
}
