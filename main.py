

from dotenv import load_dotenv
import TLSSigAPIv2
import av
import os
import traceback


load_dotenv()

secret = os.environ.get('SECRET')
sdkappid = int(os.environ.get('SDKAPPID'))
rtsp_url = os.environ.get('RTSP_URL')
room_id = os.environ.get('ROOM_ID')
user_id = int(os.environ.get('USER_ID'))


sig = TLSSigAPIv2.TLSSigAPIv2(
    sdkappid, secret)


input_ = None
output_ = None

retry = 0
while True:
    try:
        input_ = av.open(rtsp_url)
        break
    except Exception as err:
        print('rtsp pull error ')
        print(traceback.format_exc())
        retry += 1
        if retry > 3:
            raise err


usersig = sig.gen_sig(user_id, 86400)

output_url = 'rtmp://intl-rtmp.rtc.qq.com/push/{room_id}?userid={user_id}&sdkappid={sdkappid}&usersig={usersig}'.format_map({
    "room_id": room_id,
    "user_id": user_id,
    "usersig": usersig,
    "sdkappid": sdkappid
})

print("output_url: ", output_url)

output_ = av.open(output_url, 'w', format='flv')

input_video_stream = input_.streams.video[0]
input_audio_stream = input_.streams.audio[0]


output_video_stream = output_.add_stream(
    codec_name='h264')
output_audio_stream = output_.add_stream(
    codec_name='aac', rate=input_audio_stream.rate)

output_video_stream.width = input_video_stream.width
output_video_stream.height = input_video_stream.height
output_video_stream.pix_fmt = input_video_stream.pix_fmt

got_first_key_frame = False

for packet in input_.demux((input_video_stream, input_audio_stream)):
    if packet.stream.type == 'video':
        if not got_first_key_frame:
            if packet.is_keyframe:
                got_first_key_frame = True
            else:
                continue
        print('packet: ', packet.stream.type, packet.dts, packet.is_keyframe)
        packet.stream = output_video_stream
        output_.mux(packet)
    elif packet.stream.type == 'audio':
        if not got_first_key_frame:
            continue

        print('packet: ', packet.stream.type, packet.dts)
        packet.stream = output_audio_stream
        output_.mux(packet)


input_.close()
