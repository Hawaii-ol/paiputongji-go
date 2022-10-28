### 牌谱统计
***
友人场牌谱统计，揪出隐藏带恶人

### 下载
***
从[releases](https://github.com/Hawaii-ol/paiputongji-go/releases)页面下载最新的压缩包并解压

### 使用方法
***
#### 程序支持在线查询和离线查询两种方式。
直接运行程序即可连接到服务器，输入账号密码即可查询。

也可以下载网页版雀魂的数据包后离线查询：
* 在Chrome中按F12打开开发者控制台，切换到网络(Network)选项卡，然后加载雀魂界面(<https://game.maj-soul.net/1/>)
* 进入游戏后，点击牌谱，向下滚动，直到你想要结束的位置。然后在开发者控制台中切换到WS选项卡(WebSocket)，你应该会看到一个名称为gateway的WebSocket连接。右键点击它，选择**以HAR格式保存所有内容**。
* 在命令行中运行解压目录下的可执行程序paiputongji，并指定参数`--har HAR文件路径`即可。示例：

    `paiputongji /path/to/HARfile.har`

统计完成后如果没有自动弹出浏览器窗口，可以在程序目录下找到index.html打开查看。

### 源码编译
***

项目支持make构建，可用的目标如下

`make all`: 默认目标，=`make goinstall genmeta main update`

`make goinstall`: 执行`go install`命令安装编译proto文件所需的相关程序，需要网络

`make genmeta`: 根据liqi.json中的信息生成对应的liqi.proto和liqi.pb.go文件

`make main`: 生成主程序paiputongji，需要前置目标`make goinstall`和`make genmeta`

`make update`: 生成update程序，用于更新liqi.json

`make clean`: 清理build目录和liqi.proto等文件