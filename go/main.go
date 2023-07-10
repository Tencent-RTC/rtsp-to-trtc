package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/3d0c/gmf"
	"github.com/joho/godotenv"
	"github.com/tencentyun/tls-sig-api-v2-golang/tencentyun"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	secret := os.Getenv("SECRET")
	sdkappid, _ := strconv.ParseInt(os.Getenv("SDKAPPID"), 10, 0)
	rtsp_url := os.Getenv("RTSP_URL")
	room_id := os.Getenv("ROOM_ID")
	user_id := os.Getenv("USER_ID")

	log.Println("env: ", secret, sdkappid, rtsp_url, room_id, user_id)

	usersig, err := tencentyun.GenUserSig(int(sdkappid), secret, user_id, 86400*180)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(usersig)
	}

	output_url := fmt.Sprintf("rtmp://rtmp.rtc.qq.com/push/%s?userid=%s&sdkappid=%d&usersig=%s", room_id, user_id, sdkappid, usersig)

	fmt.Println(output_url)

	gmf.LogSetLevel(gmf.AV_LOG_DEBUG)

	inputCtx, err := gmf.NewInputCtx(rtsp_url)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer inputCtx.CloseInput()

	inputCtx.Dump()

	outputCtx, err := gmf.NewOutputCtxWithFormatName(output_url, "flv")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer outputCtx.CloseOutput()

	fmt.Println("===================================")

	for i := 0; i < inputCtx.StreamsCnt(); i++ {
		srcStream, err := inputCtx.GetStream(i)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		_, err = outputCtx.AddStreamWithCodeCtx(srcStream.CodecCtx())
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	outputCtx.Dump()

	if err := outputCtx.WriteHeader(); err != nil {
		fmt.Println(err.Error())
		return
	}

	// first := false
	for packet := range inputCtx.GetNewPackets() {

		fmt.Println("===================================", packet.Pts(), packet.StreamIndex())

		if packet.Dts() < 0 {
			continue
		}

		ost, err := outputCtx.GetStream(packet.StreamIndex())
		if err != nil {
			fmt.Println(err.Error())
		}

		gmf.RescaleTs(packet, ost.CodecCtx().TimeBase(), ost.TimeBase())

		//if first { //if read from rtsp ,the first packets is wrong.
		if err := outputCtx.WritePacket(packet); err != nil {
			fmt.Println(err.Error())
			return
		}
		// //}

		// // first = true
		packet.Free()
	}

}
