package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	asr "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/asr/v20190614"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tts "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tts/v20190823"
	"gopkg.in/gomail.v2"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"reflect"
	"strconv"
	"time"
	"unsafe"
)
// 账户 Model
type UserAccount struct {
	ID int `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type BlogArticle struct {
	ID int `json:"id"`
	Subject string `json:"subject"`
	Context string `json:"context"`
	Author string `json:"author"`
	Status bool `json:"status"`
}

type ASR struct {
	Response struct {
		Data struct {
			TaskID int `json:"TaskId"`
		} `json:"Data"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

type ASR2 struct {
	Response struct {
		RequestID string `json:"RequestId"`
		Data      struct {
			TaskID       int         `json:"TaskId"`
			Status       int         `json:"Status"`
			StatusStr    string      `json:"StatusStr"`
			Result       string      `json:"Result"`
			ResultDetail interface{} `json:"ResultDetail"`
			ErrorMsg     string      `json:"ErrorMsg"`
		} `json:"Data"`
	} `json:"Response"`
}

type TTS struct {
	Response struct {
		Audio     string `json:"Audio"`
		RequestID string `json:"RequestId"`
		SessionID string `json:"SessionId"`
	} `json:"Response"`
}

const (
	BASE64Table = "IJjkKLMNO567PQX12RVW3YZaDEFGbcdefghiABCHlSTUmnopqrxyz04stuvw89+/"
)

func Encode(data string) string {
	content := *(*[]byte)(unsafe.Pointer((*reflect.SliceHeader)(unsafe.Pointer(&data))))
	coder := base64.NewEncoding(BASE64Table)
	return coder.EncodeToString(content)
}

func Decode(data string) string {
	coder := base64.NewEncoding(BASE64Table)
	result, _ := coder.DecodeString(data)
	return *(*string)(unsafe.Pointer(&result))
}

func Encodefile(infile string) string{
	file, err := os.Open(infile)
	if err != nil {
		return "打开文件失败"
	}
	fileread := make([]byte,1024*50)
	n, err := file.Read(fileread)
	if err != nil {
		return "读取文件失败"
	}
	base64voice := base64.StdEncoding.EncodeToString(fileread[:n])
	return base64voice
}

func UUID() string{
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid
}

var (
	DB *gorm.DB
)
// 连接数据库
func InitMySQL()(err error){
	// 绑定数据库 (本人数据库没有密码，库名为blog)
	dsn := "root:@tcp(127.0.0.1:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"
	DB, err = gorm.Open("mysql",dsn)
	if err != nil{
		return err
	}
	err = DB.DB().Ping()
	return
}

// 如果是mail to 多个用户的话，则mailTo []string,后面的SetHeader(To,mailTo...)
func SendMail(mailTo string, subject string, body string) error {
	//定义邮箱服务器连接信息，如果是网易邮箱 pass填密码，qq邮箱填授权码

	//mailConn := map[string]string{
	//	"user": "",
	//	"pass": "",
	//	"host": "smtp.163.com",
	//	"port": "465",
	//}
	//strings.Split("", ";"),
	mailConn := map[string]string{
		"user": "",
		"pass": "",
		"host": "smtp.qq.com",
		"port": "465",
	}

	port, _ := strconv.Atoi(mailConn["port"]) //转换端口类型为int

	m := gomail.NewMessage()

	m.SetHeader("From",  m.FormatAddress(mailConn["user"], "ZonzeeLi")) //这种方式可以添加别名，即“XX官方”
	//说明：如果是用网易邮箱账号发送，以下方法别名可以是中文，如果是qq企业邮箱，以下方法用中文别名，会报错，需要用上面此方法转码
	//m.SetHeader("From", "FB Sample"+"<"+mailConn["user"]+">") //这种方式可以添加别名，即“FB Sample”， 也可以直接用<code>m.SetHeader("From",mailConn["user"])</code> 读者可以自行实验下效果
	//m.SetHeader("From", mailConn["user"])
	m.SetHeader("To", mailTo)    //发送给多个用户
	m.SetHeader("Subject", subject) //设置邮件主题
	m.SetBody("text/html", body)    //设置邮件正文

	d := gomail.NewDialer(mailConn["host"], port, mailConn["user"], mailConn["pass"])

	err := d.DialAndSend(m)
	return err

}

func main() {
	// 连接数据库
	err := InitMySQL()
	if err != nil {
		panic(err)
	}
	defer DB.Close() // 程序关闭退出数据库连接

	// 模型绑定
	DB.AutoMigrate(&UserAccount{}, &BlogArticle{})

	r := gin.Default()
	r.LoadHTMLFiles("./index.html")
	r.GET("/blog", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.POST("/upload", func(c *gin.Context) {
		// 从请求中读取文件
		f, err := c.FormFile("f1") // 从请求中获取携带的参数

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})
		} else {
			// 将读取到的文件保存到服务端本地
			//dst := fmt.Sprintf("./%s",f.Filename)
			dst := path.Join("./", f.Filename)
			_ = c.SaveUploadedFile(f, dst)
			c.JSON(http.StatusOK, gin.H{
				"status": "ok,上传成功",
			})
		}
	})
	r.POST("/signin", func(c *gin.Context) {
		// 获取form表单提交的数据
		//username := c.PostForm("username")
		//password := c.PostForm("password") //取到就返回值，取不到就返回空
		//username := c.DefaultPostForm("username","somebody")
		//password := c.DefaultPostForm("password","********")

		// 定义一个结构体获取前端数据
		var user UserAccount
		var ok bool
		user.Username, ok = c.GetPostForm("username")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "用户名注册错误",
			})
			return
		}
		user.Password, ok = c.GetPostForm("password")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "密码注册输入错误",
			})
			return
		}
		// 加密
		user.Password = Encode(user.Password)

		c.BindJSON(&user)
		// 进行判断如果用户名不存在，则进行注册，如果用户名存在则提示登陆成功
		err := DB.Where("username = ?", user.Username).First(&user).Error
		if err == gorm.ErrRecordNotFound {
			if err = DB.Create(&user).Error; err != nil {
				c.JSON(http.StatusOK, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code":   2000,
				"msg":    "success",
				"data":   user,
				"status": "ok,注册成功",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": "该用户名已存在"})

		//c.HTML(http.StatusOK,"index.html",gin.H{
		//	"Name": username,
		//	"Password": password,
		//})
	})
	r.POST("/login", func(c *gin.Context) {
		// 定义两个结构体，一个用来获取前端数据，一个用来获取数据库数据
		var verify, userinfo UserAccount
		var ok bool
		verify.Username, ok = c.GetPostForm("username")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "无效用户名",
			})
			return
		}
		verify.Password, ok = c.GetPostForm("password")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "无效密码",
			})
			return
		}
		verify.Password = Encode(verify.Password)

		// 绑定数据库
		c.BindJSON(&verify)
		c.BindJSON(&userinfo)
		// 查询前端获取的数据是否存在
		err := DB.Where("username = ?", verify.Username).First(&verify).Error
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"error": "该用户不存在，请先注册",
			})
			return
		}
		// 获取数据库中的数据，进行和前端获取的数据进行比较
		DB.Table("user_accounts").Select("password").Where("username = ?", verify.Username).First(&userinfo)
		if verify.Password == userinfo.Password {
			c.JSON(http.StatusOK, gin.H{
				"code":   2000,
				"msg":    "success",
				"data":   verify,
				"status": "ok,登录成功",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error": "用户名密码错误，请重新登录"})
	})
	r.POST("/email", func(c *gin.Context) {
		var receive, subject, content string
		receive, _ = c.GetPostForm("receive")
		subject, _ = c.GetPostForm("subject")
		content, _ = c.GetPostForm("content")

		err := SendMail(receive, subject, content)
		if err != nil {
			log.Println(err)
			fmt.Println("send fail")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": "发送成功",
		})
	})
	r.POST("/addblog", func(c *gin.Context) {
		// 后端定义博客文章结构体，从前端获取文章内容存入数据库
		var article BlogArticle
		var ok bool
		article.Subject, ok = c.GetPostForm("subject")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "文章主题无效",
			})
			return
		}
		article.Context, ok = c.GetPostForm("context")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "文章内容输入错误",
			})
			return
		}
		article.Author, ok = c.GetPostForm("author")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "作者无效",
			})
			return
		}
		// 绑定数据库
		c.BindJSON(&article)
		// 在数据库中创建该博客
		if err = DB.Create(&article).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":   2000,
			"msg":    "success",
			"status": "ok，添加成功",
		})

	})
	r.POST("/deleteblog", func(c *gin.Context) {
		var delarcticle BlogArticle

		delarcticle.Subject, _ = c.GetPostForm("delsubject")

		delarcticle.Author, _ = c.GetPostForm("delauthor")
		fmt.Println(delarcticle)

		err := DB.Where("author = ? and subject = ?", delarcticle.Author, delarcticle.Subject).Delete(&BlogArticle{}).Error
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"error": "找不到该文章",
			})
			return
		}
		c.JSON(http.StatusOK, "ok,删除成功")

	})
	r.POST("/viewblog", func(c *gin.Context) {
		var viewarcticle BlogArticle
		viewarcticle.Subject, _ = c.GetPostForm("viewsubject")
		viewarcticle.Author, _ = c.GetPostForm("viewauthor")

		c.BindJSON(&viewarcticle)

		err := DB.Where("author = ? and subject = ?", viewarcticle.Author, viewarcticle.Subject).First(&viewarcticle).Error
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{
				"error": "未找到该篇文章",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"context": viewarcticle,
		})
	})
	r.POST("/voicetotext", func(c *gin.Context) {
		// 从请求中读取文件
		f, err := c.FormFile("f2") // 从请求中获取携带的参数

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})
		} else {
			// 将读取到的文件保存到服务端本地
			//dst := fmt.Sprintf("./%s",f.Filename)
			dst := path.Join("./", f.Filename)
			_ = c.SaveUploadedFile(f, dst)
			c.JSON(http.StatusOK, gin.H{
				"status": "ok,上传成功",
			})
		}
		// 打开文件
		file, err := f.Open()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})
		}
		defer file.Close()
		// 对文件内容进行加密作为参数
		r, err := ioutil.ReadAll(file)
		voice := base64.StdEncoding.EncodeToString(r)
		// 腾讯asr部分
		credential := common.NewCredential(
			"",
			"",
		)
		cpf := profile.NewClientProfile()
		cpf.HttpProfile.Endpoint = "asr.tencentcloudapi.com"
		client, _ := asr.NewClient(credential, "", cpf)

		request := asr.NewCreateRecTaskRequest()

		request.EngineModelType = common.StringPtr("16k_zh_video")
		request.ChannelNum = common.Uint64Ptr(1)
		request.ResTextFormat = common.Uint64Ptr(0)
		request.SourceType = common.Uint64Ptr(1)
		request.Data = common.StringPtr(voice)
		// 拿到响应
		response, err := client.CreateRecTask(request)
		if _, ok := err.(*errors.TencentCloudSDKError); ok {
			fmt.Printf("An API error has returned: %s", err)
			return
		}
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s", response.ToJsonString())
		// 定义结构体绑定响应 解析json 拿到响应中的数据
		var JSON = response.ToJsonString()
		var info ASR
		err = json.Unmarshal([]byte(JSON), &info)
		if err != nil {
			fmt.Println("json Unmarshal error: ", err.Error())
			os.Exit(-1)
		}
		fmt.Println("\n",info.Response.Data.TaskID)

		////var TaskId uint64
		//result, err := gojsonq.New().FromString(response.ToJsonString()).FindR("Data.TaskId")
		////result.As(&TaskId)
		//
		//fmt.Printf("\n\n")
		//fmt.Println(result)
		time.Sleep(10*time.Second)

		request2 := asr.NewDescribeTaskStatusRequest()
		request2.TaskId = common.Uint64Ptr(uint64(info.Response.Data.TaskID))
		//request2.TaskId = common.Uint64Ptr(964926866)
		response2, err := client.DescribeTaskStatus(request2)
		if _, ok := err.(*errors.TencentCloudSDKError); ok {
			fmt.Printf("An API error has returned: %s", err)
			return
		}
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s", response2.ToJsonString())

		var Json2 = request2.ToJsonString()
		var info2 ASR2
		err = json.Unmarshal([]byte(Json2),&info2)
		if err != nil{
			fmt.Println("json Unmarshal error: ",err.Error())
			os.Exit(-1)
		}
		fmt.Println("\n",info2.Response.Data.Result)

		var article BlogArticle
		var ok bool
		article.Context = info2.Response.Data.Result
		article.Subject, ok = c.GetPostForm("voicesubject")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "文章主题无效",
			})
			return
		}
		article.Author, ok = c.GetPostForm("voiceauthor")
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"error": "文章主题无效",
			})
			return
		}

		c.BindJSON(&article)

		if err =DB.Create(&article).Error;err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK,gin.H{
			"code": 2000,
			"msg": "success",
			"ok": "语音识别成功，添加博客成功，可查找作者和主题查看",
		})
	})
	r.POST("/texttovoice", func(c *gin.Context) {
		f, err := c.FormFile("f3") // 从请求中获取携带的参数
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"error": err.Error(),
			})
		} else {

			data, err := ioutil.ReadFile(f.Filename)
			if err != nil {
				fmt.Println(err)
				return
			}

			fmt.Println(string(data))
			SessonID := UUID()

			credential := common.NewCredential(
				"",
				"",
			)
			cpf := profile.NewClientProfile()
			cpf.HttpProfile.Endpoint = "tts.tencentcloudapi.com"
			client, _ := tts.NewClient(credential, "ap-guangzhou", cpf)

			request := tts.NewTextToVoiceRequest()

			request.ModelType = common.Int64Ptr(1)
			request.SessionId = common.StringPtr(SessonID)
			request.Text = common.StringPtr(string(data))

			response, err := client.TextToVoice(request)
			if _, ok := err.(*errors.TencentCloudSDKError); ok {
				fmt.Printf("An API error has returned: %s", err)
				return
			}
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s", response.ToJsonString())

			var Json3 = response.ToJsonString()
			var info3 TTS
			err = json.Unmarshal([]byte(Json3), &info3)
			if err != nil {
				fmt.Println("json Unmarshal error: ", err.Error())
				os.Exit(-1)
			}
			fmt.Println("\n", info3.Response.Audio)

			timeNow := time.Now().Unix()
			formattime := time.Unix(timeNow,0).Format("2020-11-20-17-24-01")
			//formattime := fmt.Sprintf("%v", time.Now().Unix())

			file, err := os.Create("./tts/"+formattime+".mp3")
			if err != nil {
				fmt.Println("创建文件失败")
				fmt.Println(err)
				return
			}
			mp3, err := base64.StdEncoding.DecodeString(info3.Response.Audio)
			if err != nil {
				fmt.Println("解密返回字符串失败")
				fmt.Println(err)
				return
			}
			_, err = file.Write(mp3)
			if err != nil {
				fmt.Println("写入文件失败")
				fmt.Println(err)
				return
			}
			err = file.Close()

			c.JSON(http.StatusOK,gin.H{
				"ok": "创建成功,请返回下载",
			})
		}

	})

	r.StaticFS("/download",http.Dir("./tts"))
	r.Run(":8080")
}

