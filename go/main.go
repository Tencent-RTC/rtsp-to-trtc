package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/tencentyun/tls-sig-api-v2-golang/tencentyun"

	"github.com/asticode/go-astiav"
	"github.com/joho/godotenv"
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

	output_url := fmt.Sprintf("rtmp://intl-rtmp.rtc.qq.com/push/%s?userid=%s&sdkappid=%d&usersig=%s", room_id, user_id, sdkappid, usersig)

	fmt.Println(output_url)

	astiav.SetLogLevel(astiav.LogLevelDebug)
	astiav.SetLogCallback(func(l astiav.LogLevel, fmt, msg, parent string) {
		log.Printf("ffmpeg log: %s (level: %d)\n", strings.TrimSpace(msg), l)
	})

	pkt := astiav.AllocPacket()
	defer pkt.Free()

	inputFormatContext := astiav.AllocFormatContext()
	if inputFormatContext == nil {
		log.Fatal(errors.New("main: input format context is nil"))
	}
	defer inputFormatContext.Free()

	if err := inputFormatContext.OpenInput(rtsp_url, nil, nil); err != nil {
		log.Fatal(fmt.Errorf("main: opening input failed: %w", err))
	}
	defer inputFormatContext.CloseInput()

	// Find stream info
	if err := inputFormatContext.FindStreamInfo(nil); err != nil {
		log.Fatal(fmt.Errorf("main: finding stream info failed: %w", err))
	}

	// Alloc output format context
	outputFormatContext, err := astiav.AllocOutputFormatContext(nil, "flv", "aa.flv")
	if err != nil {
		log.Fatal(fmt.Errorf("main: allocating output format context failed: %w", err))
	}
	if outputFormatContext == nil {
		log.Fatal(errors.New("main: output format context is nil"))
	}
	defer outputFormatContext.Free()

	inputStreams := make(map[int]*astiav.Stream)
	outputStreams := make(map[int]*astiav.Stream)
	for _, istream := range inputFormatContext.Streams() {
		// Only process audio or video
		if istream.CodecParameters().MediaType() != astiav.MediaTypeAudio &&
			istream.CodecParameters().MediaType() != astiav.MediaTypeVideo {
			continue
		}

		// Add input stream
		inputStreams[istream.Index()] = istream

		if istream.CodecParameters().MediaType() == astiav.MediaTypeAudio {
			fmt.Println("index ", istream.Index(), " audio")
		}

		if istream.CodecParameters().MediaType() == astiav.MediaTypeVideo {
			fmt.Println("index ", istream.Index(), " video")
		}

		// Add stream to output format context
		ostream := outputFormatContext.NewStream(nil)
		if ostream == nil {
			log.Fatal(errors.New("main: output stream is nil"))
		}

		// Copy codec parameters
		if err = istream.CodecParameters().Copy(ostream.CodecParameters()); err != nil {
			log.Fatal(fmt.Errorf("main: copying codec parameters failed: %w", err))
		}

		// Reset codec tag
		ostream.CodecParameters().SetCodecTag(0)

		// Add output stream
		outputStreams[istream.Index()] = ostream
	}

	if err = outputFormatContext.WriteHeader(nil); err != nil {
		log.Fatal(fmt.Errorf("main: writing header failed: %w", err))
	}

	got_first_key_frame := false

	// Loop through packets
	for {
		// Read frame
		if err = inputFormatContext.ReadFrame(pkt); err != nil {
			if errors.Is(err, astiav.ErrEof) {
				break
			}
			log.Fatal(fmt.Errorf("main: reading frame failed: %w", err))
		}

		// wait fot keyframe to start
		if !got_first_key_frame {
			if !pkt.Flags().Has(astiav.PacketFlagKey) || pkt.StreamIndex() == 0 {
				pkt.Unref()
				continue
			}
			got_first_key_frame = true
		}

		fmt.Println("pkt: ", pkt.StreamIndex(), pkt.Flags().Has(astiav.PacketFlagKey))

		//Get input stream
		inputStream, ok := inputStreams[pkt.StreamIndex()]
		if !ok {
			pkt.Unref()
			continue
		}

		// Get output stream
		outputStream, ok := outputStreams[pkt.StreamIndex()]
		if !ok {
			pkt.Unref()
			continue
		}

		// Update packet
		pkt.SetStreamIndex(outputStream.Index())
		pkt.RescaleTs(inputStream.TimeBase(), outputStream.TimeBase())
		pkt.SetPos(-1)

		// Write frame
		if err = outputFormatContext.WriteInterleavedFrame(pkt); err != nil {
			log.Fatal(fmt.Errorf("main: writing interleaved frame failed: %w", err))
		}
	}

	// Write trailer
	if err = outputFormatContext.WriteTrailer(); err != nil {
		log.Fatal(fmt.Errorf("main: writing trailer failed: %w", err))
	}

}
