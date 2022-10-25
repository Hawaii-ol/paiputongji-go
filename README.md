### 牌谱统计
***
友人场牌谱统计，揪出隐藏带恶人

### 使用方法
***
1. 从[release](https://github.com/Hawaii-ol/paiputongji-go/releases)页面下载最新的压缩包并解压
2. 在Chrome中按F12打开开发者控制台，切换到网络(Network)选项卡，然后加载雀魂界面(<https://game.maj-soul.net/1/>)

3. 进入游戏后，点击牌谱，向下滚动，直到你想要结束的位置。然后在开发者控制台中切换到WS选项卡(WebSocket)，你应该会看到一个名称为gateway的WebSocket连接。右键点击它，选择**以HAR格式保存所有内容**。

4. 命令行运行解压目录下的paiputongji.exe，并传入刚刚生成的HAR文件路径作为参数，就可以自动生成牌谱统计页面了

    示例

    `paiputongji path/to/HARfile.har`

### 源码编译
***

项目支持make构建，可用的目标如下

`make goinstall`: 执行`go install`命令安装编译proto文件所需的相关程序，需要网络

`make genmeta`: 根据liqi.json中的信息生成对应的liqi.proto和liqi.pb.go文件

`make main`: 默认目标，生成可执行程序paiputongji，需要前置目标`make goinstall`和`make genmeta`

`make all`: =`make goinstall genmeta main`

`make clean`: 清理build目录和liqi.proto等文件