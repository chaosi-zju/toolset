package main

import (
	"bufio"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	_ "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"os"
	"strings"
	"time"
)

type Problem struct {
	gorm.Model
	Name    string `gorm:"column:name"`              //题目标题
	Content string `gorm:"column:content;type:text"` //题解内容
	Result  string `gorm:"column:result;type:text"`  //题目答案
	Link    string `gorm:"column:link"`              //题目链接
	Type    string `gorm:"column:type"`              //题目类别 algorithm、sql...
	SubType string `gorm:"column:sub_type"`          //题目子类别 图、树、数组...
}

type UserProblem struct {
	gorm.Model
	UserId     int       `gorm:"column:user_id;primaryKey"`    //用户ID
	ProblemId  int       `gorm:"column:problem_id;primaryKey"` //题目ID
	PickTime   time.Time `gorm:"column:pick_time"`             //选题时间
	Finished   bool      `gorm:"column:finished"`              //是否完成
	ShouldRedo bool      `gorm:"column:should_redo"`           //是否需要重做
	Times      int       `gorm:"column:times"`                 //已做次数
}

type User struct {
	gorm.Model
	Name     string `gorm:"column:name"`
	Email    string `gorm:"column:email"`
	Phone    string `gorm:"column:phone"`
	Password string `gorm:"column:password"`
	Role     string `gorm:"role"`
}

var (
	Db *gorm.DB
	fi *os.File
)

func init() {
	var err error

	viper.SetConfigFile("md_to_sql/code.yaml")
	if err = viper.ReadInConfig(); err != nil {
		panic("failed to read config:" + err.Error())
	}

	fi, err = os.Open("md_to_sql/code.md")
	if err != nil {
		panic("open file error: " + err.Error())
	}

	dsn := "root:123456@tcp(127.0.0.1:3306)/daily_problem?charset=utf8mb4&parseTime=True"
	if conn, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn}), &gorm.Config{}); err == nil {
		Db = conn
	} else {
		panic("failed to connect to mysql: " + err.Error())
	}
}

func main() {
	AddProblemFromMd()
}

func AddProblemFromMd() {

	mp := viper.GetStringMapStringSlice("problem")
	log.Infof("%+v", mp)
	problems := make(map[string]*Problem)
	for key, arr := range mp {
		for _, v := range arr {
			pro := Problem{
				Name:    v,
				Type:    "algorithm",
				SubType: key,
			}
			problems[v] = &pro
		}
	}
	log.Infof("len: %d", len(problems))

	defer fi.Close()

	br := bufio.NewReader(fi)
	curName := ""
	tempStr := ""
	lines := 0
	for {
		buf, _, err := br.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Errorf("read line error: %+v", err)
		}
		line := string(buf)

		if strings.HasPrefix(line, "## ") {
			lines++
			curName = strings.ReplaceAll(line, "## ", "")
			if _, ok := problems[curName]; !ok {
				log.Errorf("unknown problem: %s", curName)
				problems[curName] = &Problem{
					Name:    curName,
					Type:    "algorithm",
					SubType: "",
				}
			}
			br.ReadLine()
			tempStr = ""
			continue
		}
		if strings.HasPrefix(line, "[OJ链接](") {
			link := strings.ReplaceAll(line, "[OJ链接](", "")
			link = strings.ReplaceAll(link, ")", "")

			problems[curName].Link = link

			br.ReadLine()
			tempStr = ""
			continue
		}
		if line == "### 解答" {
			problems[curName].Content = tempStr
			br.ReadLine()
			tempStr = ""
			continue
		}
		if line == "<br>" {
			problems[curName].Result = tempStr
			tempStr = ""
			continue
		}

		tempStr += line + "\n"
	}

	log.Infof("len: %d", len(problems))
	log.Infof("lines: %d", lines)

	Db.AutoMigrate(Problem{})
	num := 0
	for _, pro := range problems {
		if err := Db.Create(pro).Error; err != nil {
			log.Errorf("create error: %+v", err)
		} else {
			num++
		}
	}
	Db.AutoMigrate(User{})
	Db.AutoMigrate(UserProblem{})
	log.Infof("num: %d", num)
}
