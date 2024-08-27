package storage

import (
	"github.com/gin-gonic/gin"
	"news/src/models"
	"news/src/utils"
)

// NewsService 资讯服务
type NewsService struct {
	store *Service
}

// NewNewsService creates a new NewsService
func NewNewsService() *NewsService {
	return &NewsService{
		store: NewServiceCacheFirst(),
	}
}

// HomeHandler 主页
func (s *NewsService) HomeHandler(c *utils.ApiContext) {

}

// HomeListHandler 主页完整列表
func (s *NewsService) HomeListHandler(c *utils.ApiContext) {
	category := c.Param("category")
	lang := c.Param("lang")

	list, err := s.store.GetListByCategory(category)
	if err != nil {
		c.Error(404, "资源未找到")
		return
	}

	articles := make([]articleInfo, 0, len(list))
	for _, article := range list {
		articles = append(articles, newArticleInfo(article, lang))
	}

	c.Ok(articles)
}

// HomeLinkHandler 文章链接
func (s *NewsService) HomeLinkHandler(c *utils.ApiContext) {
	token := c.Param("token")
	article, err := s.store.Get(token)
	if err != nil {
		c.Error(404, "资源未找到")
		return
	}

	c.Ok(gin.H{
		"link": article.Link,
	})
}

// HomeOriginListHandler 分类来源列表
func (s *NewsService) HomeOriginListHandler(c *utils.ApiContext) {
	category := c.PostForm("category")
	origins, err := s.store.GetOriginsByCategory(category)
	if err != nil {
		c.Error(500, "分类不存在")
		return
	}

	c.Ok(origins)
}

// NewsReadListHandler 资讯读取列表
func (s *NewsService) NewsReadListHandler(c *utils.ApiContext) {

}

// NewsOriginListHandler 资讯来源网站文章列表
func (s *NewsService) NewsOriginListHandler(c *utils.ApiContext) {
	origin := c.Param("origin")
	params := struct {
		Page     int    `form:"page,default=1" binding:"gt=0"`
		PageSize int    `form:"page_size,default=15" binding:"gt=0"`
		Lang     string `form:"lang"`
	}{}
	if err := c.ShouldBind(&params); err != nil {
		c.Error(400, "参数错误")
		return
	}

	list, count, err := s.store.GetListByOrigin(origin, params.Page, params.PageSize)
	if err != nil {
		c.Error(500, "获取文章列表失败")
		return
	}

	articles := make([]articleInfo, 0, len(list))
	for _, article := range list {
		articles = append(articles, newArticleInfo(article, params.Lang))
	}

	c.Pager(int(count), params.Page, params.PageSize, articles)
}

// NewsSearchHandler 资讯搜索
func (s *NewsService) NewsSearchHandler(c *utils.ApiContext) {

}

// 文章信息
type articleInfo struct {
	From     string `json:"from"`
	Datetime string `json:"datetime"`
	Title    string `json:"title"`
	Link     string `json:"link"`
	Author   string `json:"author"`
	Image    string `json:"image"`
	Token    string `json:"token"`
}

func newArticleInfo(article *models.Article, lang string) articleInfo {
	return articleInfo{
		From:     article.From,
		Datetime: article.PubDate.Time.Format("2006-01-02 15:04:05"),
		Title:    article.GetTitleByLang(lang),
		Link:     article.Link,
		Author:   article.Author,
		Image:    article.Image,
		Token:    article.Token,
	}
}
