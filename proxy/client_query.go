package proxy

import (
	"log"
	"strings"

	"fmt"

	"github.com/bllxi/mysql-proxy/hack"
	"github.com/bllxi/mysql-proxy/mysql"
)

func (c *Client) handleQuery(sql string) error {
	sql = strings.TrimRight(sql, ";")
	tokens := strings.FieldsFunc(sql, hack.IsSqlSep)
	if len(tokens) == 0 {
		return mysql.ErrCmdUnsupport
	}

	c.doQuery(tokens[0], sql)

	c.writeOK(nil)

	return nil
}

func (c *Client) doQuery(token string, sql string) error {
	tokenId := mysql.PARSE_TOKEN_MAP[strings.ToLower(token)]
	switch tokenId {
	case mysql.TK_ID_SELECT:
		log.Printf("query select: %s\n", sql)
		rows, err := c.server.db.Query(sql)
		if err != nil {
			return err
		}
		// var (
		// 	id           int64
		// 	arenaType    int8
		// 	platType     int8
		// 	puid         string
		// 	gameservers  string
		// 	lastGs       int
		// 	teaminfos    string
		// 	onlineMinute int64
		// 	baninfos     string
		// )
		defer rows.Close()
		for rows.Next() {
			// if err := rows.Scan(&id, &arenaType, &platType, &puid,
			// 	&gameservers, &lastGs, &teaminfos, &onlineMinute, &baninfos); err != nil {
			// 	return err
			// }
			// fmt.Printf("%v %v %v %v %v %v %v %v %v\n",
			// 	id, arenaType, platType, puid, gameservers, lastGs, teaminfos, onlineMinute, baninfos)
			var res interface{}
			if err := rows.Scan(&res); err != nil {
				return err
			}
			fmt.Printf("res: %v\n", res)
		}
	case mysql.TK_ID_DELETE:
		log.Println("query delete")
	case mysql.TK_ID_INSERT:
		log.Println("query insert")
	case mysql.TK_ID_REPLACE:
		log.Println("query replace")
	case mysql.TK_ID_UPDATE:
		log.Println("query update")
	case mysql.TK_ID_SET:
		log.Println("query set")
	case mysql.TK_ID_SHOW:
		log.Println("query show")
	case mysql.TK_ID_TRUNCATE:
		log.Println("query truncate")
	default:
		log.Println("query invalid")
	}

	return nil
}
