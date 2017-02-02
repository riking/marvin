package rss

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestFacebookFeedDataItem_Render(t *testing.T) {
	feed := FacebookFeed{}

	err := json.Unmarshal([]byte(facebookFeedExampleData), &feed)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range feed.Feed.Data {
		result := v.Render(&feed)
		b, _ := json.Marshal(result)
		fmt.Println(string(b))
	}

	b, _ := json.Marshal(feed.Feed.Data[14])
	fmt.Println(string(b))
	result := feed.Feed.Data[14].Render(&feed)
	b, _ = json.Marshal(result)
	fmt.Println(string(b))
}

const facebookFeedExampleData = `{
  "name": "42 US",
  "link": "https://www.facebook.com/42Born2CodeUS/",
  "feed": {
    "data": [
      {
        "message": "Day 2 of our weeklong Silicon Valley Survival Program featured gorgeous sunrises, 42 US, The Refiners, and award-winning business school professor Chris Haroun of Haroun Education Ventures! ESCP Europe #SBCSurvival #emdiel",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t15.0-10/s720x720/16469337_997014027066290_6525232156548005888_n.jpg?oh=6db15ae458c2bb8159a4988a1719c80f&oe=5907BAD2",
        "permalink_url": "https://www.facebook.com/StartupBasecamp/videos/997012173733142/",
        "created_time": "2017-02-02T04:52:29+0000",
        "id": "378998515534514_997012173733142"
      },
      {
        "message": "Day 2 of our week-long Silicon Valley Survival Program was the first full day and full of excitement with ESCP Europe! We visited 42 US, the innovative (and entirely free!) coding school that opened its Silicon Valley HQ in Fremont last summer, straight out of Paris. We heard from Carlos Diaz at  The Refiners about how to \"crack\" Silicon Valley. Carlos' comments on how important immigration is to the ecosystem here struck a particular cord. In the evening, the inspiring and award winning business professor Chris Haroun of Haroun Education Ventures spoke about the origins of Silicon Valley, the Venture Capital scene here, and gave his top fundraising tips!\n#SBCSurvival #EMDIEL",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/q82/p720x720/16427577_995699007197792_7782397058027337269_n.jpg?oh=0bd5bf225481fddffb042359f29a4ce2&oe=5942DE26",
        "story": "Startup Basecamp added 19 new photos.",
        "permalink_url": "https://www.facebook.com/StartupBasecamp/posts/995735057194187",
        "created_time": "2017-01-31T21:37:32+0000",
        "id": "378998515534514_995735057194187"
      },
      {
        "message": "Dear cadets, be ready for the dash (fast coding challenge) at 2:42 pm 😎",
        "full_picture": "https://external.xx.fbcdn.net/safe_image.php?d=AQDIhQ-BVgY3OniK&url=https%3A%2F%2Fmedia.giphy.com%2Fmedia%2FHTZVeK0esRjyw%2Fgiphy.gif&_nc_hash=AQDN9fW0YVkbQedo",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/446476132364481",
        "created_time": "2017-01-26T21:55:23+0000",
        "id": "284895741855855_446476132364481"
      },
      {
        "description": "While you were working on the rush01, we were thinking of you from Yosemite ;) So, how did the rush go?",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/s720x720/16265668_445807852431309_6628819483588363593_n.jpg?oh=a5a38d9cb8e5621bdf4f050f41975738&oe=59108FE4",
        "story": "Brittany Dismukes Bir shared a photo to 42 US's Timeline.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/10210571707981672",
        "created_time": "2017-01-26T00:49:10+0000",
        "id": "284895741855855_10210571707981672"
      },
      {
        "message": "How is the piscine going so far?",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/s720x720/16195430_445808669097894_3887867099506255886_n.jpg?oh=01b5f1f01750fcaeb4b4b67b8401b421&oe=59079F63",
        "story": "42 US at Yosemite National Park.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/445808669097894:0",
        "created_time": "2017-01-26T00:43:01+0000",
        "id": "284895741855855_445808669097894"
      },
      {
        "message": "While you were working on the rush01, we were thinking of you from Yosemite ;) So, how did the rush go?",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/s720x720/16265668_445807852431309_6628819483588363593_n.jpg?oh=a5a38d9cb8e5621bdf4f050f41975738&oe=59108FE4",
        "story": "42 US with Lou Guenier and Gaetan Juvin at Yosemite National Park.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/445807852431309:0",
        "created_time": "2017-01-25T22:32:23+0000",
        "id": "284895741855855_445807852431309"
      },
      {
        "message": "The power outage is finished, thanks to Kwame and Loki. Everything is back online :)",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/s720x720/16143141_445740095771418_5132015948342203262_n.jpg?oh=5547e8e52026c2544c2a7e27e04a1c55&oe=590B13C2",
        "story": "42 US added 2 new photos.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/445742535771174",
        "created_time": "2017-01-25T20:16:41+0000",
        "id": "284895741855855_445742535771174"
      },
      {
        "message": "Pedagogical meeting - Rush00 / Piscine PHP / Hercules / StarFleet 01/20/2017 :-)",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-0/p180x540/16002955_442066459472115_2651765180175500044_n.jpg?oh=0de21ed2b6ed0a03a73e415e1d214b17&oe=59408D9E",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/442066459472115:0",
        "created_time": "2017-01-20T19:35:04+0000",
        "id": "284895741855855_442066459472115"
      },
      {
        "message": "First hour of a very long night !",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/s720x720/16142742_441698892842205_8390748576518439719_n.jpg?oh=7951d70bcd6cb952fbc2062ab1479ec2&oe=5912E69F",
        "story": "42 US added 12 new photos to the album: January 2017 -  D09 — with David R. Rosa Tamsen and 10 others at 42 US.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/441698892842205",
        "created_time": "2017-01-20T04:36:10+0000",
        "id": "284895741855855_441698892842205"
      },
      {
        "message": "Are you ready to be pushed to your absolute limit? 😈\nD09 will start tomorrow for the January pisciners",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/441001906245237",
        "created_time": "2017-01-19T02:20:38+0000",
        "id": "284895741855855_441001906245237"
      },
      {
        "message": "We just opened a new check-in spot on 16th February at 6.30pm PST, you can sign up for it now\n\nRemember: the check-in is not an interview :)",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/440983802913714",
        "created_time": "2017-01-19T01:38:10+0000",
        "id": "284895741855855_440983802913714"
      },
      {
        "message": "WTF@42 - Have you tried blind coding ?",
        "full_picture": "https://external.xx.fbcdn.net/safe_image.php?d=AQBZ_b-nR_WEblgM&w=720&h=720&url=https%3A%2F%2Fi.ytimg.com%2Fvi%2FdkgLbqvTFM4%2Fmaxresdefault.jpg&cfs=1&sx=533&sy=0&sw=720&sh=720&_nc_hash=AQATEXZdJSllD3yS",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/440114066334021",
        "created_time": "2017-01-17T18:59:23+0000",
        "id": "284895741855855_440114066334021"
      },
      {
        "message": "Friday the 13th: to prevent any accident with our piscine students, we are closing all their repos, the most lucky will have it reopened...",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/15977585_437071406638287_73851051954269333_n.png?oh=9cdea274501865c14276f8c7eee365de&oe=590030CA",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/437071406638287:0",
        "created_time": "2017-01-13T18:18:40+0000",
        "id": "284895741855855_437071406638287"
      },
      {
        "description": "Paris's École 42 is reinventing education for the future",
        "full_picture": "https://external.xx.fbcdn.net/safe_image.php?d=AQAS05Rqp1l98siJ&url=https%3A%2F%2Fwi-images.condecdn.net%2Fimage%2F6qVEpXpVZOq%2Fcrop%2F1440&_nc_hash=AQB-oCSVjaO9WtYy",
        "story": "42 US shared a link.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/435328433479251",
        "created_time": "2017-01-11T00:19:54+0000",
        "id": "284895741855855_435328433479251"
      },
      {
        "description": "cc 42, Grenoble INP-Ensimag, INSA Toulouse, RWTH Aachen University, Wrocław University of Science and Technology, École normale supérieure Paris-Saclay, NC State University, Kauno „Saulės“ gimnazija, Polytechnique Montréal, Tokyo Institute of Technology, San Jose State University",
        "full_picture": "https://external.xx.fbcdn.net/safe_image.php?d=AQCdohS207iiy5jK&url=https%3A%2F%2Fwww.codingame.com%2Fblog%2Fwp-content%2Fuploads%2F2016%2F12%2Fblog-students-2-e1482336462364.jpg&_nc_hash=AQASgt3qYR_wnbxG",
        "story": "42 US shared CodinGame's post.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/435217433490351",
        "created_time": "2017-01-10T18:58:17+0000",
        "id": "284895741855855_435217433490351"
      },
      {
        "message": "Hello 42, I have been waiting for over a year, signing in on most days to get an open check-in slot. Is there ever a chance I can get in?",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/434829126862515",
        "created_time": "2017-01-10T04:27:17+0000",
        "id": "284895741855855_434829126862515"
      },
      {
        "message": "A new Piscine just begun, if you missed it get ready we have another one in April :)",
        "full_picture": "https://scontent.xx.fbcdn.net/v/l/t1.0-9/s720x720/15977444_434589086886519_3659015624041446185_n.jpg?oh=9558bf4e2bdf0dfe6c4601791b7e99a6&oe=59038BBC",
        "story": "42 US with Luis Enrique Castillo Góngora.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/434589086886519:0",
        "created_time": "2017-01-09T19:22:38+0000",
        "id": "284895741855855_434589086886519"
      },
      {
        "message": "Hi, is there any fb group for people who is going to next week piscine?",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/432650810413680",
        "created_time": "2017-01-06T16:28:43+0000",
        "id": "284895741855855_432650810413680"
      },
      {
        "message": "I want to join this...",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/432211967124231",
        "created_time": "2017-01-05T21:27:39+0000",
        "id": "284895741855855_432211967124231"
      },
      {
        "message": "We have open new check-in on January 14 and 26, you can subscribe now :)",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/426780541000707",
        "created_time": "2016-12-27T22:47:20+0000",
        "id": "284895741855855_426780541000707"
      },
      {
        "message": "What happens in vegas...",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t15.0-10/s720x720/15816264_429937877351640_1697620896446939136_n.jpg?oh=514259ef3a4e5a76031cbb7346d6759d&oe=590253E3",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/videos/429937460685015/",
        "created_time": "2016-12-23T20:00:00+0000",
        "id": "284895741855855_429937460685015"
      },
      {
        "message": "WTF@42 - #MannequinChallenge\n\n42 Silicon Valley Mannequin Challenge = teamwork makes the dream work!\n\nhttps://youtu.be/vAsCJV42AnQ",
        "full_picture": "https://external.xx.fbcdn.net/safe_image.php?d=AQCpv42k2P_Q51iS&w=720&h=720&url=https%3A%2F%2Fi.ytimg.com%2Fvi%2FvAsCJV42AnQ%2Fmaxresdefault.jpg&cfs=1&sx=129&sy=0&sw=720&sh=720&_nc_hash=AQAP_2j7pc7hmz43",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/422291061449655",
        "created_time": "2016-12-19T21:50:49+0000",
        "id": "284895741855855_422291061449655"
      },
      {
        "description": "Xavier Niel meets 42 Silicon Valley students for the first time! Thank you to the Beezwax Team for your support!",
        "message": "Xavier Niel meets 42 Silicon Valley students for the first time! Thank you to the Beezwax Team for your support!\n\nhttps://youtu.be/H9FggQG8mAk",
        "full_picture": "https://external.xx.fbcdn.net/safe_image.php?d=AQCaxNx1IjLSPO8Q&w=720&h=720&url=https%3A%2F%2Fi.ytimg.com%2Fvi%2FH9FggQG8mAk%2Fmaxresdefault.jpg&cfs=1&sx=67&sy=0&sw=720&sh=720&_nc_hash=AQCc6J9_TCyepDCw",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/422288051449956",
        "created_time": "2016-12-19T21:40:37+0000",
        "id": "284895741855855_422288051449956"
      },
      {
        "message": "Attention novices for the January piscine - the dorms are FULL. There are over 60 people on the waiting list for a room, so we will do our very best, but it may be time to look at alternative options :)",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-0/p180x540/15390903_418898051788956_685447779478108885_n.jpg?oh=5824d5a1753ac910ca73fff6e7ead28e&oe=5905A0C2",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/418898051788956:0",
        "created_time": "2016-12-14T01:17:29+0000",
        "id": "284895741855855_418898051788956"
      },
      {
        "message": "We had the delicious surprise to host Xavier Niel at our wonderful school. He was accompanied by Women Who Code. It was amazing to see how our students were so thankful and happy to have the chance to meet in-person the man who brings this experience to all of us.\n\n#42born2codeUS",
        "full_picture": "https://scontent.xx.fbcdn.net/v/t1.0-9/s720x720/15355807_411345119210916_3991001026018081776_n.jpg?oh=f91f8a0f61f6c0006cf10aa666009e75&oe=58FDAE7E",
        "story": "42 US added 2 new photos — celebrating this special day with Xavier Niel and 2 others at 42 US.",
        "permalink_url": "https://www.facebook.com/42Born2CodeUS/posts/411349662543795",
        "created_time": "2016-12-02T22:46:31+0000",
        "id": "284895741855855_411349662543795"
      }
    ],
    "paging": {
      "previous": "https://graph.facebook.com/v2.8/284895741855855/feed?fields=description,message,full_picture,story,permalink_url,created_time,id&since=1486011149&access_token=EAACEdEose0cBAL5LkBwCD4CHvuZCI1dUNQfYzuAaQekMgI5Vs4psXaTLbWZBLsmc4uUtiZBSWINKmEmqtre4rw5hHT0m2c5KCo8CiFc8fN7NsT2qteWQzXLcPaIyclywLqpWeWr5SWM8OMuQmXsenSf9cFm0VhxUliGdEimB1HfLxJGdEdHT3lPGnz5NrwZD&limit=25&__paging_token=enc_AdBu9vUHZAiNWLXHGmGipZAyFxDt5SrExWd2okwaHfj4PxrrQJKxBV6DTGM0KCLU55M8NywegxjK34aHb9gX1ZAqzJcv5UE74QUuwcq6qudJ5o99wZDZD&__previous=1",
      "next": "https://graph.facebook.com/v2.8/284895741855855/feed?fields=description,message,full_picture,story,permalink_url,created_time,id&access_token=EAACEdEose0cBAL5LkBwCD4CHvuZCI1dUNQfYzuAaQekMgI5Vs4psXaTLbWZBLsmc4uUtiZBSWINKmEmqtre4rw5hHT0m2c5KCo8CiFc8fN7NsT2qteWQzXLcPaIyclywLqpWeWr5SWM8OMuQmXsenSf9cFm0VhxUliGdEimB1HfLxJGdEdHT3lPGnz5NrwZD&limit=25&until=1480718791&__paging_token=enc_AdCQLZCIXUwsd2xXZBZAkO4ZB2N1fCFic3be3TZAgRZBTfPUdsyvZAKd73ZCpBbRbqAEb0wlOD6fhvZAmCy0CyKZBdRfnqzpbUqyP7hdSW1ZABHma7ulvKKhgZDZD"
    }
  },
  "username": "42Born2CodeUS",
  "id": "284895741855855"
}
`
