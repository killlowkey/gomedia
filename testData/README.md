# 生成测试文件
```shell
ffmpeg -i test.mp4 -vn -acodec copy test.aac
ffmpeg -i test.mp4 -an -vcodec libx265 -crf 28 test.h265
ffmpeg -i test.mp4 -an -vcodec copy -bsf:v h264_mp4toannexb test.h264
ffmpeg -i test.mp4 -c:v libx264 -c:a aac -f flv test.flv
```