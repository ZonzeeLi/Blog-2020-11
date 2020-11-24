# Blog-2020-11
该项目是基于Gin框架实现的简易博客前后端

# 具体功能：
1.实现注册登录系统，并对密码进行加密
2.博客内容的增删改查
3.发送邮件
4.语音识别（对文本内容转成音频文件并提供接口下载）
5.语音合成（对音频文件转成文本内容并添加至博文）
6.文件上传

# 使用工具：
Go（gin框架），mysql，postman，腾讯云，chrome

# 实现思路：
1.建立数据库，利用后端进行绑定，从前端获取数据，添加至数据库，利用gorm对数据库内容进行操作
2.利用gomail包实现后端发送邮件功能，可自设定主题、内容、收件人，具体数据从前端获取
3.利用腾讯云接口实现语音合成和语音识别，并建立相应结构体对返回数据进行获取
4.建立静态服务器，提供接口可以访问本地文件目录并进行下载