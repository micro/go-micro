package m3o

import (
	"go.m3o.com/address"
	"go.m3o.com/answer"
	"go.m3o.com/app"
	"go.m3o.com/avatar"
	"go.m3o.com/cache"
	"go.m3o.com/contact"
	"go.m3o.com/crypto"
	"go.m3o.com/currency"
	"go.m3o.com/db"
	"go.m3o.com/email"
	"go.m3o.com/emoji"
	"go.m3o.com/evchargers"
	"go.m3o.com/event"
	"go.m3o.com/file"
	"go.m3o.com/forex"
	"go.m3o.com/function"
	"go.m3o.com/geocoding"
	"go.m3o.com/gifs"
	"go.m3o.com/google"
	"go.m3o.com/helloworld"
	"go.m3o.com/holidays"
	"go.m3o.com/id"
	"go.m3o.com/image"
	"go.m3o.com/ip"
	"go.m3o.com/joke"
	"go.m3o.com/location"
	"go.m3o.com/movie"
	"go.m3o.com/mq"
	"go.m3o.com/news"
	"go.m3o.com/nft"
	"go.m3o.com/notes"
	"go.m3o.com/otp"
	"go.m3o.com/postcode"
	"go.m3o.com/prayer"
	"go.m3o.com/qr"
	"go.m3o.com/quran"
	"go.m3o.com/routing"
	"go.m3o.com/rss"
	"go.m3o.com/search"
	"go.m3o.com/sentiment"
	"go.m3o.com/sms"
	"go.m3o.com/space"
	"go.m3o.com/spam"
	"go.m3o.com/stock"
	"go.m3o.com/stream"
	"go.m3o.com/sunnah"
	"go.m3o.com/thumbnail"
	"go.m3o.com/time"
	"go.m3o.com/translate"
	"go.m3o.com/twitter"
	"go.m3o.com/url"
	"go.m3o.com/user"
	"go.m3o.com/vehicle"
	"go.m3o.com/weather"
	"go.m3o.com/youtube"
)

func NewClient(token string) *Client {
	return &Client{
		token: token,

		AddressService:    address.NewAddressService(token),
		AnswerService:     answer.NewAnswerService(token),
		AppService:        app.NewAppService(token),
		AvatarService:     avatar.NewAvatarService(token),
		CacheService:      cache.NewCacheService(token),
		ContactService:    contact.NewContactService(token),
		CryptoService:     crypto.NewCryptoService(token),
		CurrencyService:   currency.NewCurrencyService(token),
		DbService:         db.NewDbService(token),
		EmailService:      email.NewEmailService(token),
		EmojiService:      emoji.NewEmojiService(token),
		EvchargersService: evchargers.NewEvchargersService(token),
		EventService:      event.NewEventService(token),
		FileService:       file.NewFileService(token),
		ForexService:      forex.NewForexService(token),
		FunctionService:   function.NewFunctionService(token),
		GeocodingService:  geocoding.NewGeocodingService(token),
		GifsService:       gifs.NewGifsService(token),
		GoogleService:     google.NewGoogleService(token),
		HelloworldService: helloworld.NewHelloworldService(token),
		HolidaysService:   holidays.NewHolidaysService(token),
		IdService:         id.NewIdService(token),
		ImageService:      image.NewImageService(token),
		IpService:         ip.NewIpService(token),
		JokeService:       joke.NewJokeService(token),
		LocationService:   location.NewLocationService(token),
		MovieService:      movie.NewMovieService(token),
		MqService:         mq.NewMqService(token),
		NewsService:       news.NewNewsService(token),
		NftService:        nft.NewNftService(token),
		NotesService:      notes.NewNotesService(token),
		OtpService:        otp.NewOtpService(token),
		PostcodeService:   postcode.NewPostcodeService(token),
		PrayerService:     prayer.NewPrayerService(token),
		QrService:         qr.NewQrService(token),
		QuranService:      quran.NewQuranService(token),
		RoutingService:    routing.NewRoutingService(token),
		RssService:        rss.NewRssService(token),
		SearchService:     search.NewSearchService(token),
		SentimentService:  sentiment.NewSentimentService(token),
		SmsService:        sms.NewSmsService(token),
		SpaceService:      space.NewSpaceService(token),
		SpamService:       spam.NewSpamService(token),
		StockService:      stock.NewStockService(token),
		StreamService:     stream.NewStreamService(token),
		SunnahService:     sunnah.NewSunnahService(token),
		ThumbnailService:  thumbnail.NewThumbnailService(token),
		TimeService:       time.NewTimeService(token),
		TranslateService:  translate.NewTranslateService(token),
		TwitterService:    twitter.NewTwitterService(token),
		UrlService:        url.NewUrlService(token),
		UserService:       user.NewUserService(token),
		VehicleService:    vehicle.NewVehicleService(token),
		WeatherService:    weather.NewWeatherService(token),
		YoutubeService:    youtube.NewYoutubeService(token),
	}
}

type Client struct {
	token string

	AddressService    *address.AddressService
	AnswerService     *answer.AnswerService
	AppService        *app.AppService
	AvatarService     *avatar.AvatarService
	CacheService      *cache.CacheService
	ContactService    *contact.ContactService
	CryptoService     *crypto.CryptoService
	CurrencyService   *currency.CurrencyService
	DbService         *db.DbService
	EmailService      *email.EmailService
	EmojiService      *emoji.EmojiService
	EvchargersService *evchargers.EvchargersService
	EventService      *event.EventService
	FileService       *file.FileService
	ForexService      *forex.ForexService
	FunctionService   *function.FunctionService
	GeocodingService  *geocoding.GeocodingService
	GifsService       *gifs.GifsService
	GoogleService     *google.GoogleService
	HelloworldService *helloworld.HelloworldService
	HolidaysService   *holidays.HolidaysService
	IdService         *id.IdService
	ImageService      *image.ImageService
	IpService         *ip.IpService
	JokeService       *joke.JokeService
	LocationService   *location.LocationService
	MovieService      *movie.MovieService
	MqService         *mq.MqService
	NewsService       *news.NewsService
	NftService        *nft.NftService
	NotesService      *notes.NotesService
	OtpService        *otp.OtpService
	PostcodeService   *postcode.PostcodeService
	PrayerService     *prayer.PrayerService
	QrService         *qr.QrService
	QuranService      *quran.QuranService
	RoutingService    *routing.RoutingService
	RssService        *rss.RssService
	SearchService     *search.SearchService
	SentimentService  *sentiment.SentimentService
	SmsService        *sms.SmsService
	SpaceService      *space.SpaceService
	SpamService       *spam.SpamService
	StockService      *stock.StockService
	StreamService     *stream.StreamService
	SunnahService     *sunnah.SunnahService
	ThumbnailService  *thumbnail.ThumbnailService
	TimeService       *time.TimeService
	TranslateService  *translate.TranslateService
	TwitterService    *twitter.TwitterService
	UrlService        *url.UrlService
	UserService       *user.UserService
	VehicleService    *vehicle.VehicleService
	WeatherService    *weather.WeatherService
	YoutubeService    *youtube.YoutubeService
}
