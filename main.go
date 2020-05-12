package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/remeh/sizedwaitgroup"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/globalsign/mgo/bson"

	"github.com/globalsign/mgo"
	"github.com/prometheus/common/log"
)

var client = http.Client{}

//Package struct
type Package struct {
	Name    string `bson:"name"`
	Status  string `bson:"status"`
	Version string `bson:"version"`
}

//BaseStructPackages struct
type BaseStructPackages struct {
	Installed []Package `bson:"installed"`
}

//BaseStructDistro struct
type BaseStructDistro struct {
	Host      string `bson:"host"`
	Distro    string `bson:"distribution"`
	Release   string `bson:"release"`
	Kernel    string `bson:"kernel"`
	KernelNow string `bson:"installedkernel"`
}

//DBConnection struct
type DBConnection struct {
	URI      string
	User     string
	Password string
}

//DBQuery struct
type DBQuery struct {
	DB         string
	Collection string
	Query      string
}

//Flags
var (
	HostFlag     = kingpin.Flag("host", "Hostname(Regex)").Required().String()
	UserFlag     = kingpin.Flag("user", "User to authentificate at Database").Required().String()
	DatabaseFlag = kingpin.Flag("database", "Database that should be used").Required().String()
)

func getPackages(Host string, mongoSession *mgo.Session) BaseStructPackages {

	query := DBQuery{*DatabaseFlag, "packages", ""}
	c := mongoSession.DB(query.DB).C(query.Collection)

	pipe := c.Pipe([]bson.M{{"$match": bson.M{"host": Host}}})
	resp := []bson.M{}

	err := pipe.All(&resp)
	if err != nil {
		fmt.Println("oh")
	}

	iter := pipe.Iter()
	if errI := iter.Err(); errI != nil {
		fmt.Printf("Error while creating iterator %+v", errI)
	}

	result := BaseStructPackages{}
	for iter.Next(&result) {

	}

	if iter.Err() != nil {
		fmt.Println(iter.Err())
	}
	return result
}

//Create a mongodb Session to query the data
func mongoSession() *mgo.Session {
	conf := DBConnection{"mongodb://minventory-bs01.server.lan/" + *DatabaseFlag, *UserFlag, os.Getenv("MINVPW")}
	dialInfo, err := mgo.ParseURL(conf.URI)
	if err != nil {
		log.Errorf("Cannot parse mongodb server url: %s", err)
		os.Exit(1)
	}
	dialInfo.Timeout = 3 * time.Second
	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		log.Errorf("Cannot connect to server using url %s: %s", conf.URI, err)
		os.Exit(1)
	}
	err = session.Login(&mgo.Credential{
		Username: conf.User, Password: conf.Password,
	})
	if err != nil {
		log.Errorf("Login with supplied credentials  failed %s", err)
		os.Exit(1)
	}
	return session

}
func postRequest(swg *sizedwaitgroup.SizedWaitGroup, data BaseStructDistro, mongoSession *mgo.Session) {
	defer swg.Done()
	packagesRaw := getPackages(data.Host, mongoSession)
	packages := fmt.Sprintf("%v", packagesRaw.Installed)
	r := strings.NewReplacer(" ", ",")
	r2 := strings.NewReplacer("{", "\n")
	r3 := strings.NewReplacer("}", ",", "[", "", "]", ", ")
	packages = r3.Replace(r2.Replace(r.Replace(packages)))
	r4 := strings.NewReplacer("Debian", "debian")

	request, err := http.NewRequest("POST", "http://pocu-vuls-bs01.server.lan:5515/vuls", bytes.NewBufferString(packages))
	request.Header.Set("Content-Type", "text/plain")
	request.Header.Set("X-Vuls-OS-Family", r4.Replace(data.Distro))
	request.Header.Set("X-Vuls-Server-Name", data.Host)
	request.Header.Set("X-Vuls-OS-Release", data.Release)
	request.Header.Set("X-Vuls-Kernel-Release", data.Kernel)
	request.Header.Set("X-Vuls-Kernel-Version", data.KernelNow)

	if err != nil {
		log.Fatalln(err)
	}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatalln(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Sprintln(string(body))
}

func main() {
	kingpin.Parse()
	mongoSession := mongoSession()
	swg := sizedwaitgroup.New(100)

	query := DBQuery{*DatabaseFlag, "os", ""}
	c := mongoSession.DB(query.DB).C(query.Collection)
	pipe := c.Pipe([]bson.M{{"$match": bson.M{"host": bson.M{"$regex": *HostFlag}}}})
	resp := []bson.M{}
	err := pipe.All(&resp)
	if err != nil {
		fmt.Println("oh")
	}

	iter := pipe.Iter()
	if errI := iter.Err(); errI != nil {
		fmt.Printf("Error while creating iterator %+v", errI)
	}

	result := BaseStructDistro{}
	for iter.Next(&result) {
		swg.Add()
		go postRequest(&swg, result, mongoSession)
	}

	if iter.Err() != nil {
		fmt.Println(iter.Err())
	}
	swg.Wait()
	mongoSession.Close()

}
