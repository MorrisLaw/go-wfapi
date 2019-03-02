package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/buger/jsonparser"
	"github.com/robfig/cron"

	"github.com/Anderson-Lu/gofasion/gofasion"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/profile"
)

//current supported lang
var langid = map[string]int{
	"en": 0,
}
var platforms = [4]string{"pc", "ps4", "xb1", "swi"}
var missiontypelang map[string]interface{}
var factionslang map[string]interface{}
var locationlang map[string]interface{}
var sortiemodtypes map[string]interface{}
var sortiemoddesc map[string]interface{}
var sortiemodbosses map[string]interface{}
var sortieloc map[string]interface{}
var sortielang map[string]interface{}

var languageslang map[string]interface{}
var apidata = make([][]byte, 4)
var sortierewards = ""
var json = jsoniter.ConfigCompatibleWithStandardLibrary

var f mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func loadapidata(id1 string) (ret []byte) {
	// WF API Source
	client := &http.Client{}

	url := "https://api.warframestat.us/" + id1 + "/"
	fmt.Println("url:", url)

	req, _ := http.NewRequest("GET", url, nil)

	res, err := client.Do(req)

	if err != nil {
		fmt.Println("Errored when sending request to the server")
		return
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	_, _ = io.Copy(ioutil.Discard, res.Body)
	return body[:]
}
func main() {
	defer profile.Start(profile.MemProfile).Stop()
	gofasion.SetJsonParser(jsoniter.ConfigCompatibleWithStandardLibrary.Marshal, jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal)
	// mqtt client start
	mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)
	opts := mqtt.NewClientOptions().AddBroker("tcp://127.0.0.1:1883").SetClientID("gotrivial")
	//opts.SetKeepAlive(2 * time.Second)
	opts.SetDefaultPublishHandler(f)
	//opts.SetPingTimeout(1 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	if token := c.Subscribe("test/topic", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	//mqtt client end

	for x, v := range platforms {
		fmt.Println("x:", x)
		fmt.Println("v:", v)
		apidata[x] = loadapidata(v)
		parseAlerts(x, v, c)
		parseNews(x, v, c)
		parseSorties(x, v, c)
		parseSyndicateMissions(x, v, c)
		parseInvasions(x, v, c)
		parseCycles(x, v, c)
		PrintMemUsage()

	}
	PrintMemUsage()

	c1 := cron.New()
	c1.AddFunc("@every 1m1s", func() {

		fmt.Println("Tick")
		for x, v := range platforms {
			fmt.Println("x:", x)
			fmt.Println("v:", v)
			apidata[x] = loadapidata(v)
			parseAlerts(x, v, c)
			parseNews(x, v, c)
			parseSorties(x, v, c)
			parseSyndicateMissions(x, v, c)
			parseInvasions(x, v, c)
			parseCycles(x, v, c)

			PrintMemUsage()

		}
		/*
				parseActiveMissions(x, v, c)
				parseInvasions(x, v, c)

		}*/
	})
	c1.Start()

	PrintMemUsage()

	// just for debuging - printing  full warframe api response
	http.HandleFunc("/", sayHello)
	http.HandleFunc("/1", sayHello1)
	http.HandleFunc("/2", sayHello2)
	http.HandleFunc("/3", sayHello3)

	if err := http.ListenAndServe(":8090", nil); err != nil {
		panic(err)
	}

}
func sayHello(w http.ResponseWriter, r *http.Request) {
	message := apidata[0][:]

	w.Write([]byte(message))
}
func sayHello1(w http.ResponseWriter, r *http.Request) {
	message := apidata[1][:]

	w.Write([]byte(message))
}
func sayHello2(w http.ResponseWriter, r *http.Request) {
	message := apidata[2][:]

	w.Write([]byte(message))
}
func sayHello3(w http.ResponseWriter, r *http.Request) {
	message := apidata[3][:]

	w.Write([]byte(message))
}
func parseTests(nor int, platform string, c mqtt.Client) {
	fsion := gofasion.NewFasion(string(apidata[nor]))
	fmt.Println(fsion.Get("WorldSeed").ValueStr())
	topicf := "/wf/" + platform + "/tests"
	fmt.Println(topicf)
}

func parseAlerts(platformno int, platform string, c mqtt.Client) {
	type Alerts struct {
		ID                  string
		Started             string
		Ends                string
		MissionType         string
		MissionFaction      string
		MissionLocation     string
		MinEnemyLevel       int64
		MaxEnemyLevel       int64
		EnemyWaves          int64 `json:",omitempty"`
		RewardCredits       int64
		RewardItemMany      string `json:",omitempty"`
		RewardItemManyCount int64  `json:",omitempty"`
		RewardItem          string `json:",omitempty"`
	}
	data := apidata[platformno]
	var alerts []Alerts
	_, _, _, erralert := jsonparser.Get(data, "alerts")
	fmt.Println(erralert)
	if erralert != nil {
		fmt.Println("error alert reached")
		return
	}
	fmt.Println("alert reached")
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		id, _ := jsonparser.GetString(value, "id")
		started, _ := jsonparser.GetString(value, "activation")
		ended, _ := jsonparser.GetString(value, "expiry")
		missiontype, _ := jsonparser.GetString(value, "mission", "type")
		missionfaction, _ := jsonparser.GetString(value, "mission", "faction")
		missionlocation, _ := jsonparser.GetString(value, "mission", "node")
		minEnemyLevel, _ := jsonparser.GetInt(value, "mission", "minEnemyLevel")
		maxEnemyLevel, _ := jsonparser.GetInt(value, "mission", "maxEnemyLevel")
		enemywaves, _ := jsonparser.GetInt(value, "mission", "maxWaveNum")
		rewardcredits, _ := jsonparser.GetInt(value, "mission", "reward", "credits")
		rewarditemsmany, _ := jsonparser.GetString(value, "mission", "reward", "countedItems", "[0]", "type")
		rewarditemsmanycount, _ := jsonparser.GetInt(value, "mission", "reward", "countedItems", "[0]", "count")
		rewarditem, _ := jsonparser.GetString(value, "mission", "reward", "items", "[0]")

		w := Alerts{id, started,
			ended, missiontype,
			missionfaction, missionlocation,
			minEnemyLevel, maxEnemyLevel, enemywaves,
			rewardcredits, rewarditemsmany, rewarditemsmanycount, rewarditem}
		alerts = append(alerts, w)

	}, "alerts")

	topicf := "/wf/" + platform + "/alerts"
	messageJSON, _ := json.Marshal(alerts)
	token := c.Publish(topicf, 0, true, messageJSON)
	token.Wait()
}
func parseCycles(platformno int, platform string, c mqtt.Client) {
	type Cycles struct {
		EathID         string
		EarthEnds      string
		EarthIsDay     bool
		EarthTimeleft  string
		CetusID        string
		CetusEnds      string
		CetusIsDay     bool
		CetusIsCetus   bool
		CetusTimeleft  string
		VallisID       string
		VallisEnds     string
		VallisIsWarm   bool
		VallisTimeleft string
	}
	data := apidata[platformno]
	var cycles []Cycles
	fmt.Println("Cycles reached")
	//  Earth
	earthid, _ := jsonparser.GetString(data, "earthCycle", "id")
	earthends, _ := jsonparser.GetString(data, "earthCycle", "expiry")
	earthisday, _ := jsonparser.GetBoolean(data, "earthCycle", "isDay")
	earthtimeleft, _ := jsonparser.GetString(data, "earthCycle", "timeLeft")
	// Cetus
	cetusid, _ := jsonparser.GetString(data, "cetusCycle", "id")
	cetusends, _ := jsonparser.GetString(data, "cetusCycle", "expiry")
	cetusisday, _ := jsonparser.GetBoolean(data, "cetusCycle", "isDay")
	cetusiscetus, _ := jsonparser.GetBoolean(data, "cetusCycle", "isCetus")
	cetustimeleft, _ := jsonparser.GetString(data, "cetusCycle", "timeLeft")
	// Vallis
	vallisid, _ := jsonparser.GetString(data, "vallisCycle", "id")
	vallisends, _ := jsonparser.GetString(data, "vallisCycle", "expiry")
	vallisiswarm, _ := jsonparser.GetBoolean(data, "vallisCycle", "isDay")
	vallistimeleft, _ := jsonparser.GetString(data, "vallisCycle", "timeLeft")

	w := Cycles{earthid, earthends, earthisday, earthtimeleft,
		cetusid, cetusends, cetusisday, cetusiscetus, cetustimeleft,
		vallisid, vallisends, vallisiswarm, vallistimeleft}
	cycles = append(cycles, w)

	topicf := "/wf/" + platform + "/cycles"
	messageJSON, _ := json.Marshal(cycles)
	token := c.Publish(topicf, 0, true, messageJSON)
	token.Wait()
}
func parseNews(platformno int, platform string, c mqtt.Client) {
	type Newsmessage struct {
		LanguageCode string
		Message      string
	}
	type News struct {
		ID       string
		Message  string
		URL      string
		Date     string
		priority bool
		Image    string
	}
	data := apidata[platformno]
	_, _, _, ernews := jsonparser.Get(data, "news")
	if ernews != nil {
		fmt.Println("error ernews reached")
		return
	}
	var news []News

	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		_, _, _, translationerr := jsonparser.Get(value, "translations", "en")
		if translationerr != nil {
			return
		}
		image := "http://n9e5v4d8.ssl.hwcdn.net/uploads/e0b4d18d3330bb0e62dcdcb364d5f004.png"
		message := ""
		id, _ := jsonparser.GetString(value, "id")

		message, _ = jsonparser.GetString(value, "translations", "en")
		url, _ := jsonparser.GetString(value, "link")
		image, _ = jsonparser.GetString(value, "imageLink")
		date, _ := jsonparser.GetString(value, "date")
		/**/
		priority, _ := jsonparser.GetBoolean(value, "priority")
		w := News{ID: id, Message: message, URL: url, Date: date, Image: image, priority: priority}
		news = append(news, w)
		topicf := "/wf/" + platform + "/news"
		messageJSON, _ := json.Marshal(news)
		token := c.Publish(topicf, 0, true, messageJSON)
		token.Wait()

	}, "news")
}

func parseSorties(platformno int, platform string, c mqtt.Client) {
	type Sortievariant struct {
		MissionType     string
		MissionMod      string
		MissionModDesc  string
		MissionLocation string
	}
	type Sortie struct {
		ID       string
		Started  string
		Ends     string
		Boss     string
		Faction  string
		Reward   string
		Variants []Sortievariant
		Active   bool
	}
	fmt.Println("reached sortie start")
	data := apidata[platformno]
	sortieactive, sortieerr := jsonparser.GetBoolean(data, "sortie", "active")
	if sortieerr != nil || sortieactive != true {
		fmt.Println("reached sortie error")

		return
	}
	fmt.Println("reached sortie start2")

	var sortie []Sortie
	id, _ := jsonparser.GetString(data, "sortie", "id")
	started, _ := jsonparser.GetString(data, "sortie", "activation")
	ended, _ := jsonparser.GetString(data, "sortie", "expiry")
	boss, _ := jsonparser.GetString(data, "sortie", "boss")
	faction, _ := jsonparser.GetString(data, "sortie", "faction")
	reward, _ := jsonparser.GetString(data, "sortie", "rewardPool")
	var variants []Sortievariant

	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		mtype, _ := jsonparser.GetString(value, "missionType")
		mmod, _ := jsonparser.GetString(value, "modifier")
		mmoddesc, _ := jsonparser.GetString(value, "modifierDescription")
		mloc, _ := jsonparser.GetString(value, "node")

		variants = append(variants, Sortievariant{
			MissionType:     mtype,
			MissionMod:      mmod,
			MissionModDesc:  mmoddesc,
			MissionLocation: mloc,
		})
	}, "sortie", "variants")
	active, _ := jsonparser.GetBoolean(data, "sortie", "active")
	w := Sortie{ID: id, Started: started,
		Ends: ended, Boss: boss, Faction: faction, Reward: reward, Variants: variants,
		Active: active}
	sortie = append(sortie, w)

	topicf := "/wf/" + platform + "/sorties"
	messageJSON, _ := json.Marshal(sortie)
	token := c.Publish(topicf, 0, true, messageJSON)
	token.Wait()

}

func parseSyndicateMissions(platformno int, platform string, c mqtt.Client) {
	type SyndicateJobs struct {
		Jobtype        string
		Rewards        []string
		MinEnemyLevel  int64
		MaxEnemyLevel  int64
		StandingReward []int64
	}
	type SyndicateMissions struct {
		ID        string
		Started   string
		Ends      string
		Syndicate string
		Jobs      []SyndicateJobs
	}
	data := apidata[platformno]
	var syndicates []SyndicateMissions
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		syndicatecheck, _ := jsonparser.GetString(value, "syndicate")
		if syndicatecheck != "Ostrons" && syndicatecheck != "Solaris United" {
			return
		}
		id, _ := jsonparser.GetString(value, "id")
		started, _ := jsonparser.GetString(value, "activation")
		ended, _ := jsonparser.GetString(value, "expiry")
		syndicate, _ := jsonparser.GetString(value, "syndicate")
		var jobs []SyndicateJobs
		jsonparser.ArrayEach(value, func(value1 []byte, dataType jsonparser.ValueType, offset int, err error) {
			jobtype, _ := jsonparser.GetString(value1, "type")
			rewards := make([]string, 0)
			jsonparser.ArrayEach(value1, func(reward []byte, dataType jsonparser.ValueType, offset int, err error) {
				rewards = append(rewards, string(reward))

			}, "rewardPool")

			minEnemyLevel, _ := jsonparser.GetInt(value1, "enemyLevels", "[0]")
			maxEnemyLevel, _ := jsonparser.GetInt(value1, "enemyLevels", "[1]")
			standing1, _ := jsonparser.GetInt(value1, "standingStages", "[0]")
			standing2, _ := jsonparser.GetInt(value1, "standingStages", "[1]")
			standing3, _ := jsonparser.GetInt(value1, "standingStages", "[2]")
			jobs = append(jobs, SyndicateJobs{
				Jobtype:        jobtype,
				Rewards:        rewards,
				MinEnemyLevel:  minEnemyLevel,
				MaxEnemyLevel:  maxEnemyLevel,
				StandingReward: []int64{standing1, standing2, standing3},
			})
		}, "jobs")

		w := SyndicateMissions{
			ID:        id,
			Started:   started,
			Ends:      ended,
			Syndicate: syndicate,
			Jobs:      jobs}
		syndicates = append(syndicates, w)
	}, "syndicateMissions")

	topicf := "/wf/" + platform + "/syndicates"
	messageJSON, _ := json.Marshal(syndicates)
	token := c.Publish(topicf, 0, true, messageJSON)
	token.Wait()

}
func parseInvasions(platformno int, platform string, c mqtt.Client) {
	type Invasion struct {
		ID                  string
		Location            string
		MissionType         string
		Completed           bool
		Started             string
		VsInfested          bool
		AttackerRewardItem  string `json:",omitempty"`
		AttackerRewardCount int64  `json:",omitempty"`
		AttackerMissionInfo string `json:",omitempty"`
		DefenderRewardItem  string `json:",omitempty"`
		DefenderRewardCount int64  `json:",omitempty"`
		DefenderMissionInfo string `json:",omitempty"`
		Completion          float64
	}

	data := apidata[platformno]
	invasioncheck, _, _, _ := jsonparser.Get(data, "invasions")
	if len(invasioncheck) == 0 {
		return
	}
	var invasions []Invasion
	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		iscomplete, _ := jsonparser.GetBoolean(value, "completed")
		if iscomplete != true {
			attackeritem := ""
			attackeritemcount := int64(0)
			defenderitem := ""
			defenderitemcount := int64(0)
			id, _ := jsonparser.GetString(value, "id")
			started, _ := jsonparser.GetString(value, "activation")
			location, _ := jsonparser.GetString(value, "node")
			missiontype, _ := jsonparser.GetString(value, "desc")
			completed, _ := jsonparser.GetBoolean(value, "completed")
			vsinfested, _ := jsonparser.GetBoolean(value, "vsInfestation")
			jsonparser.ArrayEach(value, func(value1 []byte, dataType jsonparser.ValueType, offset int, err error) {
				attackeritem, _ = jsonparser.GetString(value1, "type")
				attackeritemcount, _ = jsonparser.GetInt(value1, "count")
			}, "attackerReward")
			attackerfaction, _ := jsonparser.GetString(value, "attackingFaction")
			jsonparser.ArrayEach(value, func(value1 []byte, dataType jsonparser.ValueType, offset int, err error) {
				defenderitem, _ = jsonparser.GetString(value1, "type")
				defenderitemcount, _ = jsonparser.GetInt(value1, "count")
			}, "defenderReward")

			defenderfaction, _ := jsonparser.GetString(value, "defendingFaction")
			completion, _ := jsonparser.GetFloat(value, "completion")
			w := Invasion{id, location, missiontype, completed, started, vsinfested,
				attackeritem, attackeritemcount, attackerfaction,
				defenderitem, defenderitemcount, defenderfaction, completion}
			invasions = append(invasions, w)
		}
	}, "invasions")

	topicf := "/wf/" + platform + "/invasions"
	messageJSON, _ := json.Marshal(invasions)
	token := c.Publish(topicf, 0, true, messageJSON)
	token.Wait()
}

/*
func parseActiveMissions(platformno int, platform string, c mqtt.Client) {
	type ActiveMissions struct {
		ID          string
		Started     int
		Ends        int
		Region      int
		Node        string
		MissionType string
		Modifier    string
	}
	data := &apidata[platformno]
	fsion := gofasion.NewFasion(*data)
	var mission []ActiveMissions
	lang := string("en")
	ActiveMissionsarray := fsion.Get("ActiveMissions").Array()
	fmt.Println(len(ActiveMissionsarray))

	for _, v := range ActiveMissionsarray {
		id := v.Get("_id").Get("$oid").ValueStr()
		started := v.Get("Activation").Get("$date").Get("$numberLong").ValueInt() / 1000
		ended := v.Get("Expiry").Get("$date").Get("$numberLong").ValueInt() / 1000
		region := v.Get("Region").ValueInt()
		node := v.Get("Node").ValueStr()
		missiontype := v.Get("MissionType").ValueStr()
		modifier := v.Get("Modifier").ValueStr()

		w := ActiveMissions{
			ID:          id,
			Started:     started,
			Ends:        ended,
			Region:      region,
			Node:        node,
			MissionType: missiontype,
			Modifier:    modifier,
		}
		mission = append(mission, w)
	}
	fmt.Println(len(mission))
	topicf := "/wf/" + platform + "/missions"
	messageJSON, _ := json.Marshal(mission)
	token := c.Publish(topicf, 0, true, messageJSON)
	token.Wait()
}
*/
func calcCompletion(count int, goal int, attacker string) (complete float32) {
	y := float32((1 + float32(count)/float32(goal)))
	x := float32(y * 50)
	if attacker == "Infested" {
		x = float32(y * 100)

	}
	//fmt.Println(y)
	return x
}

// PrintMemUsage - only for debug
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
