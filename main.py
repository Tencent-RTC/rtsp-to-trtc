

from dotenv import load_dotenv
import TLSSigAPIv2
import av
import os
import traceback


load_dotenv()

secret = os.environ.get('SECRET')
sdkappid = os.environ.get('SDKAPPID')
rtsp_url = os.environ.get('RTSP_URL')
room_id = os.environ.get('ROOM_ID')
user_id = os.environ.get('USER_ID')


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

output_url = 'rtmp://test12.rtc.qq.com/push/{room_id}?userid={user_id}&sdkappid={sdkappid}&usersig={usersig}&use_number_room_id=1'.format_map({
    "room_id": room_id,
    "user_id": user_id,
    "usersig": usersig,
    "sdkappid": sdkappid
})

output_ = av.open(output_url, 'w')

input_video_stream = input_.streams.video[0]
input_audio_stream = input_.streams.audio[0]

output_video_stream = output_.add_stream(template=input_video_stream)
output_audio_stream = output_.add_stream(template=input_audio_stream)

for packet in input_.demux((input_video_stream, input_audio_stream)):
    if packet.stream.type == 'video':
        packet.stream = output_video_stream
    elif packet.stream.type == 'audio':
        packet.stream = output_audio_stream
    output_.mux(packet)
