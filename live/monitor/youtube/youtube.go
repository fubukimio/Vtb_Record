package youtube

import (
	"fmt"
	"github.com/fzxiao233/Vtb_Record/config"
	"github.com/fzxiao233/Vtb_Record/live/interfaces"
	"github.com/fzxiao233/Vtb_Record/live/monitor/base"
	"github.com/fzxiao233/Vtb_Record/live/monitor/bilibili"
	. "github.com/fzxiao233/Vtb_Record/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"regexp"
	"sync"
	"time"
)

type yfConfig struct {
	IsLive bool
	Title  string
	Target string
}
type Youtube struct {
	base.BaseMonitor
	yfConfig
	usersConfig config.UsersConfig
}

func (y *Youtube) getVideoInfo() error {
	url := "https://www.youtube.com" + "/channel/" + y.usersConfig.TargetId + "/live"
	htmlBody, err := y.Ctx.HttpGet(url, map[string]string{})
	if err != nil {
		return err
	}
	re, _ := regexp.Compile(`ytplayer.config\s*=\s*([^\n]+?});`)
	result := re.FindSubmatch(htmlBody)
	if len(result) < 2 {
		return fmt.Errorf("youtube cannot find js_data")
	}
	jsonYtConfig := result[1]
	playerResponse := gjson.GetBytes(jsonYtConfig, "args.player_response")
	if !playerResponse.Exists() {
		return fmt.Errorf("youtube cannot find player_response")
	}
	videoDetails := gjson.Get(playerResponse.String(), "videoDetails")
	if !playerResponse.Exists() {
		return fmt.Errorf("youtube cannot find videoDetails")
	}
	IsLive := videoDetails.Get("isLive").Bool()
	if !IsLive {
		return err
	} else {
		y.Title = videoDetails.Get("title").String()
		y.Target = "https://www.youtube.com/watch?v=" + videoDetails.Get("videoId").String()
		y.IsLive = IsLive
		return nil
	}
	//return nil, nil
	//log.Printf("%+v", y)
}

type YoutubePoller struct {
	LivingUids map[string]bilibili.LiveInfo
	lock       sync.Mutex
}

var U2bPoller YoutubePoller

func (y *YoutubePoller) parseBaseStatus(rawPage string) ([]string, error) {
	livingUids := make([]string, 0)

	re, _ := regexp.Compile(`var\s*ytInitialGuideData\s*=\s*([^\n]+?});`)
	result := re.FindStringSubmatch(rawPage)
	if len(result) < 1 {
		return livingUids, fmt.Errorf("youtube cannot find ytInitialGuideData")
	}
	addItem := func(itm gjson.Result) {
		isLive := itm.Get("guideEntryRenderer.badges.liveBroadcasting")
		if isLive.Bool() == false {
			return
		}

		browsed_id := itm.Get("guideEntryRenderer.navigationEndpoint.browseEndpoint.browseId")
		//title := itm.Get("guideEntryRenderer.title")

		livingUids = append(livingUids, browsed_id.String())
	}

	jsonYtConfig := result[1]
	jsonParsed := gjson.Parse(jsonYtConfig)
	items1 := jsonParsed.Get("items")
	for _, item := range items1.Array() {
		items2 := item.Get("guideSubscriptionsSectionRenderer.items")
		if !items2.Exists() {
			continue
		}
		for _, item2 := range items2.Array() {
			if !item2.Get("guideCollapsibleEntryRenderer").Exists() {
				addItem(item2)
			} else {
				item3 := item2.Get("guideCollapsibleEntryRenderer.expandableItems")
				for _, item4 := range item3.Array() {
					if item4.Get("guideEntryRenderer.badges").Exists() {
						addItem(item4)
					}
				}
			}
		}
	}

	log.Tracef("Parsed base uids: %s", livingUids)
	return livingUids, nil
}

func (y *YoutubePoller) parseSubscStatus(rawPage string) (map[string]bilibili.LiveInfo, error) {
	livingUids := make(map[string]bilibili.LiveInfo)

	re, _ := regexp.Compile(`\["ytInitialData"\]\s*=\s*([^\n]+?});`)
	result := re.FindStringSubmatch(rawPage)
	if len(result) < 1 {
		//y.LivingUids = livingUids
		return livingUids, fmt.Errorf("youtube cannot find js_data")
	}
	jsonYtConfig := result[1]
	items := gjson.Get(jsonYtConfig, "contents.twoColumnBrowseResultsRenderer.tabs.0.tabRenderer.content.sectionListRenderer.contents.0.itemSectionRenderer.contents.0.shelfRenderer.content.gridRenderer.items")
	itemArr := items.Array()
	for _, item := range itemArr {
		style := item.Get("gridVideoRenderer.badges.0.metadataBadgeRenderer.style")

		if style.String() == "BADGE_STYLE_TYPE_LIVE_NOW" {
			channelId := item.Get("gridVideoRenderer.shortBylineText.runs.0.navigationEndpoint.browseEndpoint.browseId")
			//title := item.Get("gridVideoRenderer.shortBylineText.runs.0.text")
			videoId := item.Get("gridVideoRenderer.videoId")
			//video_thumbnail := item.Get("gridVideoRenderer.thumbnail.thumbnails.0.url")
			videoTitle := item.Get("gridVideoRenderer.title.simpleText")
			//upcomingEventData := item.Get("gridVideoRenderer.upcomingEventData")

			livingUids[channelId.String()] = bilibili.LiveInfo{
				Title:         videoTitle.String(),
				StreamingLink: "https://www.youtube.com/watch?v=" + videoId.String(),
			}
		}

	}

	//y.LivingUids = livingUids
	log.Tracef("Parsed uids: %s", livingUids)
	return livingUids, nil
}

func (y *YoutubePoller) getLiveStatus() error {
	var err error
	ctx := base.GetCtx("Youtube")
	//mod := interfaces.GetMod("Youtube")
	apihosts := []string{
		"https://nameless-credit-7c9e.misty.workers.dev",
		"https://delicate-cherry-9564.vtbrecorder1.workers.dev",
		"https://plain-truth-41a9.vtbrecorder2.workers.dev",
		"https://snowy-shape-95ae.vtbrecorder3.workers.dev",
		"https://shiny-cake-4abc.vtbrecorder4.workers.dev",
		"https://snowy-sound-d33b.vtbrecorder5.workers.dev",
		"https://lingering-art-8c8e.vtbrecorder6.workers.dev",
		"https://muddy-forest-b1aa.vtbrecorder7.workers.dev",
		"https://delicate-smoke-8638.vtbrecorder8.workers.dev",
		"https://weathered-cherry-4472.vtbrecorder9.workers.dev",
		"https://raspy-surf-ce6b.vtbrecorder10.workers.dev",
		"https://square-violet-9579.vtbrecorder11.workers.dev",
	}

	livingUids := make(map[string]bilibili.LiveInfo)
	/*
		rawPageBase, err := ctx.HttpGet(
			RandChooseStr(apihosts),
			map[string]string{})
		if err != nil {
			return err
		}
		pagebase := string(rawPageBase)
		baseUids, err := y.parseBaseStatus(pagebase)
		if err != nil {
			return err
		}
		for _, chanId := range baseUids {
			if _, ok := livingUids[chanId]; !ok {
				liveinfo, err := getVideoInfo(ctx, RandChooseStr(apihosts), chanId)
				if liveinfo != nil {
					livingUids[chanId] = *liveinfo
				} else {
					log.WithError(err).Warnf("Failed to get live info for channel %s", chanId)
				}
			}
		}
	*/

	rawPage, err := ctx.HttpGet(
		RandChooseStr(apihosts)+"/feed/subscriptions/",
		map[string]string{})
	if err != nil {
		return err
	}
	page := string(rawPage)
	subscUids, err := y.parseSubscStatus(page)
	if err != nil {
		return err
	}
	for k, v := range subscUids {
		livingUids[k] = v
	}

	y.LivingUids = livingUids
	return nil
}

func (y *YoutubePoller) GetStatus() error {
	return y.getLiveStatus()
}

func (y *YoutubePoller) StartPoll() error {
	err := y.GetStatus()
	if err != nil {
		return err
	}
	mod := base.GetMod("Youtube")
	_interval, ok := mod.ExtraConfig["PollInterval"]
	interval := time.Duration(config.Config.CriticalCheckSec) * time.Second
	if ok {
		interval = time.Duration(_interval.(float64)) * time.Second
	}
	go func() {
		for {
			time.Sleep(interval)
			err := y.GetStatus()
			if err != nil {
				log.WithError(err).Warnf("Error during polling GetStatus")
			}
		}
	}()
	return nil
}

func (y *YoutubePoller) IsLiving(uid string) *bilibili.LiveInfo {
	y.lock.Lock()
	if y.LivingUids == nil {
		err := y.StartPoll()
		if err != nil {
			log.WithError(err).Warnf("Failed to poll from youtube")
		}
	}
	y.lock.Unlock()
	info, ok := y.LivingUids[uid]
	if ok {
		return &info
	} else {
		return nil
	}
}

func (b *Youtube) getVideoInfoByPoll() error {
	ret := U2bPoller.IsLiving(b.usersConfig.TargetId)
	b.IsLive = ret != nil
	if !b.IsLive {
		return nil
	}

	b.Target = ret.StreamingLink
	b.Title = ret.Title
	return nil
}

func (y *Youtube) CreateVideo(usersConfig config.UsersConfig) *interfaces.VideoInfo {
	if !y.yfConfig.IsLive {
		return &interfaces.VideoInfo{}
	}
	v := &interfaces.VideoInfo{
		Title:       y.Title,
		Date:        GetTimeNow(),
		Target:      y.Target,
		Provider:    "Youtube",
		UsersConfig: usersConfig,
	}
	return v
}
func (y *Youtube) CheckLive(usersConfig config.UsersConfig) bool {
	y.usersConfig = usersConfig
	err := y.getVideoInfo()
	if err != nil {
		y.IsLive = false
	}
	if !y.IsLive {
		base.NoLiving("Youtube", usersConfig.Name)
	}
	return y.yfConfig.IsLive
}

//func (y *Youtube) StartMonitor(usersConfig UsersConfig) {
//	if y.CheckLive(usersConfig) {
//		ProcessVideo(y.createVideo(usersConfig))
//	}
//}
