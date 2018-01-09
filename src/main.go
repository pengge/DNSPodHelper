package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

/*
公共参数 (*号必选)
	* login_token: 用于鉴权的 API Token
	format: {json,xml} 返回的数据格式，默认为xml，建议用json
	lang: {en,cn} 返回的错误语言，默认为en，建议用cn
	error_on_empty: {yes,no} 没有数据时是否返回错误，默认为yes，建议用no
	user_id: 用户的ID，仅代理接口需要，用户接口不需要提交此参数
*/
type PublicParams struct {
	LoginToken   string `json:"login_token"`
	Format       string `json:"format"`
	Lang         string `json:"lang"`
	ErrorOnEmpty string `json:"error_on_empty"`
	UserId       string `json:"user_id"`
}

/*
记录参数
	domain 域名
	domain_id 域名ID
	sub_domain 主机记录
	record_type 记录类型
	record_line 记录线路
	value 记录值
	mx {1-20} MX优先级
	ttl {1-604800} TTL，范围1-604800
	status [“enable”, “disable”]
*/
type Record struct {
	Domain     string `json:"domain"`
	DomainId   string `json:"domain_id"`
	SubDomain  string `json:"sub_domain"`
	RecordId   string `json:"record_id"`
	RecordType string `json:"record_type"`
	RecordLine string `json:"record_line"`
	Value      string `json:"value"`
	MX         int    `json:"mx"`
	TTL        int    `json:"ttl"`
	Status     string `json:"status"`
}

func NewDefaultRecord() Record {
	return Record{
		RecordType: "A",
		RecordLine: "默认",
		TTL:        600,
	}
}

/*
结构体转为url参数
*/
func Struct2Values(objects ...interface{}) (value url.Values) {
	values := url.Values{}
	for i := range objects {
		obj := objects[i]
		objType := reflect.TypeOf(obj)
		objValue := reflect.ValueOf(obj)
		for i := 0; i < objType.NumField(); i++ {
			fieldName := objType.Field(i).Tag.Get("json")
			fieldValue := objValue.Field(i)
			switch fieldValue.Type().Kind() {
			case reflect.Int:
				values.Add(fieldName, string(strconv.FormatInt(objValue.Field(i).Int(), 10)))
			case reflect.String:
				values.Add(fieldName, objValue.Field(i).String())
			}
		}
	}
	return values
}

/**
获取帐户信息
	https://dnsapi.cn/User.Detail
请求内容
	公共参数
*/
func UserDetail(publicParams PublicParams) error {
	apiUrl := "https://dnsapi.cn/User.Detail"
	values := Struct2Values(publicParams)
	resp := HttpPost(apiUrl, values)
	respJsonMap, err := JSON2Map([]byte(resp))
	if err != nil {

	}
	if respJsonMap["status"] != "1" {
		return errors.New("API_Token验证失败，请确认Token是否正确")
	}
	return nil
}

/*
添加记录
	API: https://dnsapi.cn/Record.Create
请求内容
	公共参数
	* domain_id 或 domain: 分别对应域名ID和域名, 提交其中一个即可
	* value: 记录值, 如 IP:200.200.200.200, CNAME: cname.dnspod.com., MX: mail.dnspod.com., 必选
	* record_type: 记录类型，通过API记录类型获得，大写英文，比如：A
	* mx: {1-20} MX优先级, 当记录类型是 MX 时有效，范围1-20
	sub_domain: 主机记录, 如 www，如果不传，默认为 @
	record_line: 记录线路，通过API记录线路获得，中文，比如：默认
	record_line_id: 线路的ID，通过API记录线路获得，英文字符串，比如：‘10=1’ 【record_line 和 record_line_id 二者传其一即可，系统优先取 record_line_id】
	ttl: {1-604800} TTL，范围1-604800，不同等级域名最小值不同,
	status: [“enable”, “disable”]，记录初始状态，默认为”enable”，如果传入”disable”，解析不会生效，也不会验证负载均衡的限制，可选
	weight: 权重信息，0到100的整数。仅企业 VIP 域名可用，0 表示关闭，留空或者不传该参数，表示不设置权重信息
*/
func CreateRecord(publicParams PublicParams, record *Record) (resp string, err error) {
	apiUrl := "https://dnsapi.cn/Record.Create"
	values := Struct2Values(publicParams, *record)
	resp = HttpPost(apiUrl, values)
	jsonMap, _ := JSON2Map([]byte(resp))
	if statusMap := jsonMap["status"].(map[string]interface{}); statusMap["code"] != "1" {
		return resp, errors.New(statusMap["message"].(string))
	}
	recordMap := jsonMap["record"].(map[string]interface{})
	record.RecordId = recordMap["id"].(string)
	log.Printf("新纪录 %s.%s 添加成功", record.Domain, record.SubDomain)
	return resp, nil
}

/*
更新记录
	API: https://dnsapi.cn/Record.Modify
请求参数
	公共参数
	* domain_id 或 domain: 分别对应域名ID和域名, 提交其中一个即可
	* record_id: 记录ID
	* record_type: 记录类型，通过API记录类型获得，大写英文，比如：A
	* record_line: 记录线路，通过API记录线路获得，中文，比如：默认
    * value: 记录值, 如 IP:200.200.200.200, CNAME: cname.dnspod.com., MX: mail.dnspod.com.，必选
	mx: {1-20} MX优先级, 当记录类型是 MX 时有效，范围1-20
	sub_domain: 主机记录, 如 www，可选，如果不传，默认为 @
	record_line_id: 线路的ID，通过API记录线路获得，英文字符串，比如：‘10=1’ 【record_line 和 record_line_id 二者传其一即可，系统优先取 record_line_id】
	ttl: {1-604800} TTL，范围1-604800，不同等级域名最小值不同
	status: [“enable”, “disable”]，记录状态，默认为”enable”，如果传入”disable”，解析不会生效，也不会验证负载均衡的限制，可选
	weight: 权重信息，0到100的整数，可选。仅企业 VIP 域名可用，0 表示关闭，留空或者不传该参数，表示不设置权重信息
*/
func UpdateRecord(publicParams PublicParams, record Record) (resp string, err error) {
	apiUrl := "https://dnsapi.cn/Record.Modify"
	values := Struct2Values(publicParams, record)
	resp = HttpPost(apiUrl, values)
	jsonMap, _ := JSON2Map([]byte(resp))
	if statusMap := jsonMap["status"].(map[string]interface{}); statusMap["code"] != "1" {
		return resp, errors.New(statusMap["message"].(string))
	}
	log.Printf("更新记录 %s.%s 成功", record.SubDomain, record.Domain)
	return resp, nil
}

/*
查找指定记录，并将record_id写入record，用于dnspod记录信息同步到本地。
	API: https://dnsapi.cn/Record.List
请求参数
	公共参数
    * domain_id 或 domain, 分别对应域名ID和域名, 提交其中一个即可
	offset 记录开始的偏移，第一条记录为 0，依次类推，可选
	length 共要获取的记录的数量，比如获取20条，则为20，可选
	sub_domain 子域名，如果指定则只返回此子域名的记录，可选
	keyword，搜索的关键字，如果指定则只返回符合该关键字的记录，可选
*/
func SyncRecords(publicParams PublicParams, record *Record) (resp string, err error) {
	apiUrl := "https://dnsapi.cn/Record.List"
	values := Struct2Values(publicParams)
	values.Add("domain", record.Domain)
	values.Add("sub_domain", record.SubDomain)
	resp = HttpPost(apiUrl, values)
	respJsonMap, _ := JSON2Map([]byte(resp))

	//查找记录
	//查到了，写入record_id
	//如果没有查到记录，请求添加记录
	if respJsonMap["status"].(map[string]interface{})["code"] == "1" {
		recordsArray := respJsonMap["records"].([]interface{})
		recordId := recordsArray[0].(map[string]interface{})["id"].(string)
		record.RecordId = recordId
	} else {
		record.Value = string(GetIP())
		resp, err := CreateRecord(publicParams, record)
		if err != nil {
			log.Printf("记录 %s.%s 添加失败", &record.SubDomain, &record.Domain)
		}
		return resp, err
	}
	return resp, nil
}

/*
通用函数，用于发送POST请求
*/
func HttpPost(url string, values url.Values) (resp string) {
	client := http.DefaultClient
	body := values.Encode()
	request, requestParseErr := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if requestParseErr != nil {
		log.Println("请求解析失败")
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "DNSPodHelper/1.0 (panja@foxmail.com)")
	response, err := client.Do(request)
	if err != nil {
		log.Panicf("请求失败! 请确保为网络通畅！%s", url)
	}
	respBytes, _ := ioutil.ReadAll(response.Body)

	return string(respBytes)
}

//获取外网IP地址
func GetIP() []byte {
	rAddr, e := net.ResolveTCPAddr("tcp", "ns1.dnspod.net:6666")
	if e != nil {
		log.Println(e.Error())
	}
	conn, _ := net.DialTCP("tcp", nil, rAddr)
	bytes, _ := ioutil.ReadAll(conn)
	return bytes
}

/*
配置参数
	Setting 常规参数
	PublicParam 公共参数
	Records 需要更新的记录数组
*/
type Configuration struct {
	Setting      map[string]interface{}
	PublicParams PublicParams
	Records      []Record
	FilePath     string
}

/*
JSON转Map
*/
func JSON2Map(jsonBytes []byte) (jsonMap map[string]interface{}, err error) {
	if unmarshalError := json.Unmarshal(jsonBytes, &jsonMap); unmarshalError != nil {
		err = errors.New("解析失败，请确认JSON格式是否正确")
		return nil, err
	}
	return jsonMap, nil
}

//解析配置文件
func ParseConfigFile(configFilePath string) (config Configuration, err error) {
	configBytes, readError := ioutil.ReadFile(configFilePath)
	if readError != nil {
		return config, errors.New("无法读取配置文件，请确认文件路径是否正确")
	}

	configMap, parseError := JSON2Map(configBytes)
	if parseError != nil {
		return config, errors.New("无法读取配置文件，请确认参数格式是否正确")
	}

	//解析常规参数
	config.Setting = make(map[string]interface{})
	settingMap := configMap["Setting"].(map[string]interface{})

	//解析ApiToken
	apiToken := settingMap["api_token"]
	id := apiToken.(map[string]interface{})["id"].(string)
	token := apiToken.(map[string]interface{})["token"].(string)
	publicParams := PublicParams{
		LoginToken: id + "," + token,
		Format:     "json",
	}

	//Token验证失败
	if validErr := UserDetail(publicParams); validErr != nil {
		log.Println(validErr.Error())
		os.Exit(-1)
	}

	//解析Record数组
	recordsMap := configMap["Records"].([]interface{})
	records := make([]Record, len(recordsMap))
	for i, recordParam := range recordsMap {
		recordParamMap := recordParam.(map[string]interface{})
		record := NewDefaultRecord()
		marshal, _ := json.Marshal(recordParamMap)
		json.Unmarshal(marshal, &record)
		SyncRecords(publicParams, &record)
		records[i] = record
	}

	config.FilePath = configFilePath
	config.PublicParams = publicParams
	config.Records = records

	return config, nil
}

/*
周期性更新记录
*/
func (configuration *Configuration) UpdateRecordsInCycle(duration time.Duration) {
	for {
		var timer *time.Timer
		timer = time.NewTimer(duration)
		log.Println("---开始更新记录---")
		ip := string(GetIP())
		log.Printf("当前公网IP为 %s", ip)
		for _, record := range configuration.Records {
			record.Value = ip
			UpdateRecord(configuration.PublicParams, record)
		}
		log.Println("---更新结束---")
		<-timer.C
	}
}

func main() {
	//解析CLI参数
	//配置文件地址
	var configFilePath string
	var intervalSeconds uint
	flag.StringVar(&configFilePath, "c", "", "配置文件路径")
	flag.UintVar(&intervalSeconds, "t", 600, "记录更新间隔时间（秒）")
	flag.Parse()

	//读取配置文件
	var config Configuration
	config, err := ParseConfigFile(configFilePath)
	if err != nil {
		flag.Usage()
		os.Exit(-1)
	}

	//周期性更新记录
	config.UpdateRecordsInCycle(time.Duration(intervalSeconds) * time.Second)

	//异常恢复
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
}
