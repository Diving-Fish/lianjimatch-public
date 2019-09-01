
package controller

import (
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/kataras/iris"
	"github.com/kataras/iris/core/errors"
	"io/ioutil"
	"net/http"
	"sort"
	"time"
)

type JSON map[string]interface{}

type Team struct {
	ID		int		`gorm:"AUTO_INCREMENT"`
	Name	string
	Group	int
	Password string
	Qq		int64
}

type Player struct {
	ID		int
	Name	string
	TeamID	int
	Qq		int64
	IsLeader bool
	Position int
}

type Process struct {
	ID			int		`gorm:"AUTO_INCREMENT"`
	TeamID		int
	PlayerID	int
	Point		int
	Round		int
	Process		int
}

type Group struct {
	Area		int
	Round		int
	Team1		int
	Team2		int
	Team3		int
}

var db *gorm.DB

// Admin Routers

func getNowProcess(teamId int, round int) (nowProcess int, point int) {
	var processes []Process
	db.Where("team_id = ? and round = ?", teamId, round).Find(&processes)
	nowProcess = -1
	point = 100000
	for _, process := range processes {
		if process.Process > nowProcess {
			nowProcess = process.Process
			point = process.Point
		}
	}
	return
}

func in(ele string, arr []string) bool {
	for _, v := range arr {
		if v == ele {
			return true
		}
	}
	return false
}

func toStringArray(arr []interface{}) (sarr []string) {
	for _, v := range arr {
		sarr = append(sarr, v.(string))
	}
	return
}

func GetAllStatus(ctx iris.Context) {
	round, _ := ctx.URLParamInt("round")
	var groups []Group
	var ret []map[string]interface{}
	db.Where("round = ?", round).Find(&groups)
	result, _ := http.Get("http://localhost:5000/get_now_info")
	body, _ := ioutil.ReadAll(result.Body)
	if string(body) == "busy" {
		_, _ = ctx.JSON(JSON{
			"busy": 1,
		})
		return
	}
	var info map[string]interface{}
	_ = json.Unmarshal(body, &info)
	playing := info["playing"].([]interface{})
	ready := toStringArray(info["ready"].([]interface{}))
	for _, group := range groups {
		var ele map[string]interface{}
		nowProcess, point1 := getNowProcess(group.Team1, round)
		nowProcess, point2 := getNowProcess(group.Team2, round)
		nowProcess, point3 := getNowProcess(group.Team3, round)
		nextProcess := nowProcess + 1
		var player1, player2, player3 Player
		var playerNames []string
		flag := true
		db.Where("team_id = ? and position = ?", group.Team1, toPos(nextProcess)).First(&player1)
		db.Where("team_id = ? and position = ?", group.Team2, toPos(nextProcess)).First(&player2)
		db.Where("team_id = ? and position = ?", group.Team3, toPos(nextProcess)).First(&player3)
		playerNames = []string {player1.Name, player2.Name, player3.Name}
		sort.Strings(playerNames)
		if len(playing) == 0 {
			flag = false
		}
		for _, p := range playing {
			flag = true
			s := toStringArray(p.([]interface{}))
			sort.Strings(s)
			for i := 0; i < len(s); i++ {
				if s[i] != playerNames[i] {
					flag = false
					break
				}
			}
			if flag {
				ele = JSON{
					"status": 1,
					"process": nextProcess,
					"players": []map[string]interface{} {
						JSON{ "id": player1.ID, "name": player1.Name, "team_id": player1.TeamID, "point": point1 },
						JSON{ "id": player2.ID, "name": player2.Name, "team_id": player2.TeamID, "point": point2 },
						JSON{ "id": player3.ID, "name": player3.Name, "team_id": player3.TeamID, "point": point3 },
					},
				}
				ret = append(ret, ele)
				break
			}
		}
		//if nextProcess == 4 {
		if nextProcess == 6 {
			db.Where("team_id = ? and position = ?", group.Team1, 2).First(&player1)
			db.Where("team_id = ? and position = ?", group.Team2, 2).First(&player2)
			db.Where("team_id = ? and position = ?", group.Team3, 2).First(&player3)
			ele = JSON{
				"status": 3,
				"process": 6, // 4
				"players": []map[string]interface{} {
					JSON{ "id": player1.ID, "name": player1.Name, "team_id": player1.TeamID, "point": point1 },
					JSON{ "id": player2.ID, "name": player2.Name, "team_id": player2.TeamID, "point": point2 },
					JSON{ "id": player3.ID, "name": player3.Name, "team_id": player3.TeamID, "point": point3 },
				},
			}
			ret = append(ret, ele)
		} else if !flag {
			db.Where("team_id = ? and position = ?", group.Team1, nextProcess).First(&player1)
			db.Where("team_id = ? and position = ?", group.Team2, nextProcess).First(&player2)
			db.Where("team_id = ? and position = ?", group.Team3, nextProcess).First(&player3)
			ready1 := in(player1.Name, ready)
			ready2 := in(player2.Name, ready)
			ready3 := in(player3.Name, ready)
			status := 0
			if ready1 && ready2 && ready3 {
				status = 2
			}
			ele = JSON{
				"status": status,
				"process": nextProcess,
				"players": []map[string]interface{} {
					JSON{ "id": player1.ID, "name": player1.Name, "team_id": player1.TeamID, "point": point1, "ready": ready1 },
					JSON{ "id": player2.ID, "name": player2.Name, "team_id": player2.TeamID, "point": point2, "ready": ready2 },
					JSON{ "id": player3.ID, "name": player3.Name, "team_id": player3.TeamID, "point": point3, "ready": ready3 },
				},
			}
			ret = append(ret, ele)
		}
	}
	_, _ = ctx.JSON(ret)
}

func AdminHandler(ctx iris.Context) {
	if ctx.GetHeader("Authorization") == "fake_pwd" {
		ctx.Next()
	} else {
		ctx.StatusCode(iris.StatusUnauthorized)
		return
	}
}

func AdminAuth(ctx iris.Context) {
	_, _ = ctx.JSON(map[string]interface{}{
		"status": 200,
	})
}

func DeleteTeamAdmin(ctx iris.Context) {
	teamId, _ := ctx.URLParamInt("team_id")
	team := Team{}
	var players []Player
	db.Where("team_id = ?", teamId).Find(&players)
	for _, player := range players {
		db.Delete(&player)
	}
	db.Where("id = ?", teamId).First(&team)
	db.Delete(&team)
	_, _ = ctx.JSON(JSON{
		"msg": "success",
	})
}

func FetchAllPlayers(ctx iris.Context) {
	var players []Player
	db.Find(&players)
	var teams []Team
	db.Find(&teams)
	teamMap := map[int][]Player{

	}
	for _, player := range players {
		teamMap[player.TeamID] = append(teamMap[player.TeamID], player)
	}
	var result []map[string]interface{}
	teams = append(teams, Team{
		ID: 0,
	})
	for _, team := range teams {
		var players []map[string]interface{}
		for _, player := range teamMap[team.ID] {
			players = append(players, map[string]interface{} {
				"id": player.ID,
				"name": player.Name,
				"leader": player.IsLeader,
			})
		}
		result = append(result, map[string]interface{} {
			"team_id": team.ID,
			"qq": team.Qq,
			"password": team.Password,
			"group": team.Group,
			"team_name": team.Name,
			"players": players,
		})
	}
	_, _ = ctx.JSON(result)
}

// Team Routers

func AuthHandler(ctx iris.Context) {
	id := GetId(ctx.GetHeader("Authorization"))
	if id == -1 {
		ctx.StatusCode(iris.StatusUnauthorized)
		_, _ = ctx.JSON(JSON{
			"msg": "not login",
		})
		return
	}
	ctx.Values().Set("team_id", id)
	ctx.Next()
}

func Auth(ctx iris.Context) {
	teamId, _ := ctx.Values().GetInt("team_id")
	_, _ = ctx.JSON(JSON{
		"team_id": teamId,
	})
}

func SetPosition(ctx iris.Context) {
	var positionJson map[string]interface{}
	err := ctx.ReadJSON(&positionJson)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "json error",
		})
		return
	}
	teamId, _ := ctx.Values().GetInt("team_id")
	var players []Player
	db.Where("team_id = ?", teamId).Find(&players)
	for _, player := range players {
		player.Position = -1
		for _, p := range positionJson["positions"].([]interface{}) {
			position := p.(map[string]interface{})
			if player.Name == position["name"].(string) {
				player.Position = int(position["position"].(float64))
			}
		}
		db.Save(&player)
	}
	_, _ = ctx.JSON(JSON{
		"msg": "success",
	})
}

func DeleteTeam(ctx iris.Context) {
	teamId, _ := ctx.Values().GetInt("team_id")
	team := Team{}
	var players []Player
	db.Where("team_id = ?", teamId).Find(&players)
	for _, player := range players {
		db.Delete(&player)
	}
	db.Where("id = ?", teamId).First(&team)
	db.Delete(&team)
	_, _ = ctx.JSON(JSON{
		"msg": "success",
	})
}

func AddPlayer(ctx iris.Context) {
	playerJson := JSON{}
	teamId, _ := ctx.Values().GetInt("team_id")
	err := ctx.ReadJSON(&playerJson)
	if err != nil || playerJson["id"] == nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "json error",
		})
		return
	}
	var pl []Player
	db.Where("team_id = ?", teamId).Find(&pl)
	isLeader := false
	if len(pl) == 0 {
		isLeader = true
	} else if teamId != 200 && len(pl) >= 5 {
		_, _ = ctx.JSON(JSON{
			"full": 1,
			"msg": "team is full",
		})
		return
	}
	player := Player{}
	db.First(&player, int(playerJson["id"].(float64)))
	if player.ID != 0 {
		_, _ = ctx.JSON(JSON{
			"reg": 1,
			"msg": "player has been reg",
		})
		return
	}
	name, err := findPlayer(int(playerJson["id"].(float64)))
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "can't find player",
		})
		return
	}
	player = Player{
		ID:     int(playerJson["id"].(float64)),
		Name:   name,
		TeamID: teamId,
		IsLeader: isLeader,
	}
	db.Create(&player)
	_, _ = ctx.JSON(JSON{
		"msg": "success",
	})
}

// Public Routers

func SendProcess(ctx iris.Context) {
	str := ctx.URLParam("name")
	point, _ := ctx.URLParamInt("point")
	round, _ := ctx.URLParamInt("round")
	player := Player{}
	db.Where("name = ?", str).First(&player)
	var processes []Process
	db.Where("team_id = ? and round = ?", player.TeamID, round).Find(&processes)
	process := Process{
		Process: len(processes),
		TeamID: player.TeamID,
		PlayerID: player.ID,
		Point: point,
		Round: round,
	}
	db.Create(&process)
}

func GetAllStats(ctx iris.Context) {
	round, _ := ctx.URLParamInt("round")
	var groups []Group
	var ret []map[string]interface{}
	db.Where("round = ?", round).Find(&groups)
	for _, group := range groups {
		var teams []map[string]interface{}
		teams = append(teams, getStatInfo(group.Team1, round))
		teams = append(teams, getStatInfo(group.Team2, round))
		teams = append(teams, getStatInfo(group.Team3, round))
		ret = append(ret, JSON{
			"area": group.Area,
			"teams": teams,
		})
	}
	_, _ = ctx.JSON(ret)
}

func GetStatsByGroup(ctx iris.Context) {
	round, _ := ctx.URLParamInt("round")
	var ret []map[string]interface{}
	for i := 0; i < 4; i++ {
		ret = append(ret, JSON{
			"area": i,
			"teams": []JSON{},
		})
	}
	var teams []Team
	db.Where("`group` >= 0 and `group` <= 3").Find(&teams)
	for _, team := range teams {
		ret[team.Group]["teams"] = append(ret[team.Group]["teams"].([]JSON), getStatInfo(team.ID, round))
	}
	_, _ = ctx.JSON(ret)
}

func GetAllGroups(ctx iris.Context) {
	round, _ := ctx.URLParamInt("round")
	var groups []Group
	var ret []map[string]interface{}
	db.Where("round = ?", round).Find(&groups)
	for _, group := range groups {
		var teams []map[string]interface{}
		teams = append(teams, getTeamInfo(group.Team1))
		teams = append(teams, getTeamInfo(group.Team2))
		teams = append(teams, getTeamInfo(group.Team3))
		ret = append(ret, JSON{
			"area": group.Area,
			"teams": teams,
		})
	}
	_, _ = ctx.JSON(ret)
}

func Login(ctx iris.Context) {
	teamJson := JSON{}
	err := ctx.ReadJSON(&teamJson)
	if err != nil || teamJson["id"] == nil || teamJson["password"] == nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "json error",
		})
		return
	}
	team := Team{}
	db.Find(&team, int(teamJson["id"].(float64)))
	if team.ID == 0 {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "can't find team",
		})
		return
	}
	if team.Password == teamJson["password"].(string) {
		_, _ = ctx.JSON(JSON{
			"msg": "login successfully",
			"token": BuildToken(team.ID),
		})
	} else {
		ctx.StatusCode(iris.StatusUnauthorized)
		_, _ = ctx.JSON(JSON{
			"msg": "wrong pwd",
		})
		return
	}
}

func CreateTeam(ctx iris.Context) {
	teamJson := JSON{}
	err := ctx.ReadJSON(&teamJson)
	if err != nil || teamJson["name"] == nil || teamJson["password"] == nil || teamJson["qq"] == nil || teamJson["id"] == nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "json error",
		})
		return
	}
	team := Team{}
	db.Where("name = ?", teamJson["name"].(string)).First(&team)
	if team.ID != 0 {
		_, _ = ctx.JSON(JSON{
			"msg": "duplicate",
		})
		return
	}
	team = Team{
		Name: teamJson["name"].(string),
		Group: 0,
		Password: teamJson["password"].(string),
		Qq:		int64(teamJson["qq"].(float64)),
	}
	player := Player{}
	db.First(&player, int(teamJson["id"].(float64)))
	if player.ID != 0 {
		_, _ = ctx.JSON(JSON{
			"reg": 1,
			"msg": "player has been reg",
		})
		return
	}
	name, err := findPlayer(int(teamJson["id"].(float64)))
	if err != nil {
		_, _ = ctx.JSON(JSON{
			"msg": "can't find player",
		})
		return
	}
	db.Create(&team)
	db.Where("name = ?", teamJson["name"].(string)).First(&team)
	player = Player{
		ID:       int(teamJson["id"].(float64)),
		Name:     name,
		TeamID:   team.ID,
		IsLeader: true,
	}
	db.Create(&player)
	_, _ = ctx.JSON(JSON{
		"team_id": team.ID,
	})
}

func GetPlayers(ctx iris.Context) {
	id, _ := ctx.URLParamInt("id")
	var pl []Player
	db.Where("team_id = ?", id).Find(&pl)
	var players []JSON
	for _, p := range pl {
		players = append(players, JSON{
			"id": p.ID,
			"name": p.Name,
			"leader": p.IsLeader,
			"position": p.Position,
		})
	}
	_, _ = ctx.JSON(JSON{
		"players": players,
	})
}

func SingleSign(ctx iris.Context) {
	signJson := JSON{}
	err := ctx.ReadJSON(&signJson)
	if err != nil || signJson["id"] == nil || signJson["qq"] == nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "json error",
		})
		return
	}
	id := int(signJson["id"].(float64))
	qq := int64(signJson["qq"].(float64))
	player := Player{}
	db.Where("id = ?", id).First(&player)
	if player.ID != 0 {
		_, _ = ctx.JSON(JSON{
			"reg": 1,
			"msg": "player has been registered",
		})
		return
	}
	player.Name, err = findPlayer(id)
	if err != nil {
		ctx.StatusCode(iris.StatusBadRequest)
		_, _ = ctx.JSON(JSON{
			"msg": "can't find player",
		})
	}
	player.ID = id
	player.Qq = qq
	player.TeamID = 0
	player.IsLeader = false
	db.Create(&player)
	_, _ = ctx.JSON(JSON{
		"msg": "single sign successfully",
	})
}

func QueryPlayer(ctx iris.Context) {
	id, _ := ctx.URLParamInt("id")
	player := Player{}
	db.Where("id = ?", id).First(&player)
	if player.ID != 0 {
		_, _ = ctx.JSON(map[string]interface{} {
			"reg": 1,
		})
	} else {
		name, err := findPlayer(id)
		if err != nil {
			_, _ = ctx.JSON(map[string]interface{} {
				"err": 1,
			})
		} else {
			_, _ = ctx.JSON(map[string]interface{} {
				"name": name,
			})
		}
	}
}

// Functions

func getStatInfo(id int, round int) map[string]interface{} {
	team := Team{}
	db.Where("id = ?", id).First(&team)
	var pro []Process
	db.Where("team_id = ? and round = ?", id, round).Find(&pro)
	var scores []int
	// for i := 0; i < 4; i++ {
	for i := 0; i < 6; i++ {
		for _, p := range pro {
			if p.Process == i {
				scores = append(scores, p.Point)
			}
		}
	}
	return JSON{
		"team_id": team.ID,
		"team_name": team.Name,
		"scores": scores,
	}
}

func getTeamInfo(id int) map[string]interface{} {
	team := Team{}
	db.Where("id = ?", id).First(&team)
	var pl []Player
	db.Where("team_id = ?", id).Find(&pl)
	var players []JSON
	for _, p := range pl {
		players = append(players, JSON{
			"id": p.ID,
			"name": p.Name,
			"leader": p.IsLeader,
			"position": p.Position,
		})
	}
	return JSON{
		"team_id": team.ID,
		"team_name": team.Name,
		"players": players,
	}
}

func findPlayer(id int) (name string, err error) {
	resp, _ := http.Get(fmt.Sprintf("http://localhost:5000/get_username/%d", id))
	b, _ := ioutil.ReadAll(resp.Body)
	name = string(b)
	if len(name) >= 30 || name == "获取角色信息出错" || name == "busy" {
		return "", errors.New("can't find")
	}
	return name, nil
}

func BuildToken(id int) string {
	claims := make(jwt.MapClaims)
	claims["id"] = id
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	claims["iat"] = time.Now().Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("lqynb"))
	return tokenString
}

func GetId(tokenString string) int {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte("lqynb"), nil
	})
	if err != nil {
		return -1
	} else if !token.Valid {
		return -1
	} else {
		return int(token.Claims.(jwt.MapClaims)["id"].(float64))
	}
}

func toPos(process int) int {
	//if process >= 3 {
	//	return process - 1
	//}
	//return process
	return process / 2
}

func init() {
	var err error
	db, err = gorm.Open("mysql", "fake_user:fake_pwd@tcp(localhost:3306)/ljmatch")
	if err != nil {
		fmt.Println(err)
		return
	}
	if !db.HasTable(&Team{}) {
		db.CreateTable(&Team{})
	}
	if !db.HasTable(&Player{}) {
		db.CreateTable(&Player{})
	}
	if !db.HasTable(&Process{}) {
		db.CreateTable(&Process{})
	}
	if !db.HasTable(&Group{}) {
		db.CreateTable(&Group{})
	}
}
