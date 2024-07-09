#以shell举例
#1在当前目录下创建一个overlay2目录并且切换到该目录
mkdir overlay2 && cd overlay2
#2创建container-layer、image-layer、work、mnt目录
mkdir container-layer image-layer work mnt
#3像container-layer、image-layer写入文件
echo "I am container-layer" > container-layer/container-layer.txt
echo "I am image-layer" > image-layer/image-layer.txt
#4将container-layer、image-layer挂载到mnt目录下 可读层为image-layer、可写层为container-layer 工作目录为work
sudo mount -t overlay -o lowerdir=image-layer,upperdir=container-layer,workdir=work none mnt
#5像mnt下的image-layer.txt写入内容
echo "write to image-layer" >> mnt/image-layer.txt
#6查看mnt/image-layer.txt发现多了个内容ls container-layer/
cat mnt/image-layer.txt
#7查看image-layer/image-layer.txt 没有新增的内容
cat image-layer/image-layer.txt
#8查看container-layer 发现多了一个image-layer.txt文本并且有新增的内容
cat container-layer/image-layer.txt
#总结 修改可读层就是将可读的数据复制一份到可写层，并在上面进行修改 cow机制