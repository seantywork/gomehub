package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"

	"gocv.io/x/gocv"

	"log"

	"github.com/wimspaargaren/yolov3"

	"github.com/gorilla/websocket"
	"github.com/seantywork/gomehub/pkg/dbquery"
)

func turn_on_gui_with_video() {

	webcam, _ := gocv.OpenVideoCapture(0)
	window := gocv.NewWindow("Hello")
	img := gocv.NewMat()

	for {
		webcam.Read(&img)
		window.IMShow(img)
		window.WaitKey(1)
	}

}

func in_your_face() {

	deviceID := 0

	// open webcam
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer webcam.Close()

	// open display window
	window := gocv.NewWindow("Face Detect")
	defer window.Close()

	// prepare image matrix
	img := gocv.NewMat()
	defer img.Close()

	// color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load("data/haarcascade_frontalface_default.xml") {
		fmt.Println("Error reading cascade file: data/haarcascade_frontalface_default.xml")
		return
	}

	fmt.Printf("start reading camera device: %v\n", deviceID)
	for {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("cannot read device %v\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}

		// detect faces
		rects := classifier.DetectMultiScale(img)
		fmt.Printf("found %d faces\n", len(rects))

		// draw a rectangle around each face on the original image
		for _, r := range rects {
			gocv.Rectangle(&img, r, blue, 3)
		}

		// show the image in the window, and wait 1 millisecond
		window.IMShow(img)
		window.WaitKey(1)
	}

}

func webcam_yolo() {

	var (
		yolov3WeightsPath = "vendor/yolo/yolo.weights"
		yolov3ConfigPath  = "vendor/yolo/yolo.cfg"
		cocoNamesPath     = "vendor/yolo/coco.names"
	)

	yolonet, err := yolov3.NewNet(yolov3WeightsPath, yolov3ConfigPath, cocoNamesPath)
	if err != nil {

		log.Fatalf("unable to create yolo net: %s\n", err.Error())

		return

	}

	// Gracefully close the net when the program is done
	defer func() {
		err := yolonet.Close()
		if err != nil {

			log.Fatalf("unable to gracefully close yolo net: %s\n", err.Error())

			return

		}
	}()

	videoCapture, err := gocv.OpenVideoCapture(0)
	if err != nil {

		log.Fatalf("unable to start video capture: %s\n", err.Error())

		return

	}

	window := gocv.NewWindow("Result Window")
	defer func() {
		err := window.Close()
		if err != nil {

			log.Fatalf("unable to close window: %s\n", err.Error())

			return
		}
	}()

	frame := gocv.NewMat()
	defer func() {
		err := frame.Close()
		if err != nil {

			log.Fatalf("unable to close image: %s\n", err.Error())

			return
		}
	}()

	for {
		if ok := videoCapture.Read(&frame); !ok {

			log.Println("unable to read stream")

		}
		if frame.Empty() {
			continue
		}
		detections, err := yolonet.GetDetections(frame)

		if err != nil {

			log.Fatalf("unable to retrieve prediction: %s\n", err.Error())

		}

		detections_len := len(detections)

		for i := 0; i < detections_len; i++ {

			fmt.Printf("detection: %s, confidence: %f\n", detections[i].ClassName, detections[i].Confidence)

		}

		yolov3.DrawDetections(&frame, detections)

		window.IMShow(frame)
		window.WaitKey(1)
	}

}

func print_bits() {

	var bits int64

	bits = 8 << 20

	fmt.Println(bits)
}

type RT_REQ_DATA struct {
	Command string `json:"command"`
	Data    string `json:"data"`
}

type RT_RESP_DATA struct {
	Status string `json:"status"`
	Data   string `json:"data"`
}

func sorrylinus_roundtrip() {

	// this is localhost actually

	u := "ws://feebdaed.xyz:3000"

	c, _, err := websocket.DefaultDialer.Dial(u, nil)

	if err != nil {
		log.Fatal("dial:", err)
	}

	defer c.Close()

	go func() {
		for {

			resp_data := RT_RESP_DATA{}
			err := c.ReadJSON(&resp_data)
			if err != nil {
				log.Println("read:", err)
				return
			}

			data_b, _ := json.Marshal(resp_data)

			log.Printf("recv: %s", string(data_b))
		}
	}()

	auth_data := RT_REQ_DATA{
		Command: "none",
		Data:    "seantywork@gmail.com:letsshareitwiththewholeuniverse",
	}

	c.WriteJSON(auth_data)

	var cmdin string

	for {

		cmdin = ""

		req_data := RT_REQ_DATA{}

		fmt.Printf("$ data: ")

		fmt.Scanln(&cmdin)

		req_data.Command = "ROUNDTRIP"
		req_data.Data = cmdin

		c.WriteJSON(req_data)

	}

}
func test_editorjs_parse() {

	file_b, _ := os.ReadFile("./test.json")

	tlist, err := dbquery.GetAssociateMediaKeysForEditorjsSrc(file_b)

	if err != nil {

		fmt.Println(err.Error())

		return

	}

	fmt.Println(tlist)

}

func main() {

	//	turn_on_gui_with_video()

	// in_your_face()

	//webcam_yolo()

	// print_bits()

	//sorrylinus_roundtrip()

	test_editorjs_parse()

}
