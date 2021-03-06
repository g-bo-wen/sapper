package manager

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"

	"github.com/juju/errors"
	"github.com/zssky/log"

	"github.com/dearcode/sapper/meta"
)

// result 必需是一个指向切片的指针
func query(table, where, sort, order string, offset, count int, result interface{}) (int, error) {
	rt := reflect.TypeOf(result)
	rv := reflect.ValueOf(result).Elem()

	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Slice {
		return 0, fmt.Errorf("result type must be ptr to slice, recv:%v", rt.Kind())
	}

	fs := rt.Elem().Elem()
	if fs.NumField() == 0 {
		return 0, fmt.Errorf("result not found field")
	}

	dt := strings.Split(table, ",")[0]

	fields := bytes.NewBuffer([]byte{})
	for i := 0; i < fs.NumField(); i++ {
		name := fs.Field(i).Tag.Get("db")
		if name == "" {
			name = strings.ToLower(fs.Field(i).Name)
		}
		if !strings.Contains(name, ".") {
			fields.WriteString(dt)
			fields.WriteString(".")
		}
		fields.WriteString(name)
		fields.WriteString(", ")
	}

	fields.Truncate(fields.Len() - 2)

	bs := bytes.NewBufferString("select ")
	bs.WriteString(fields.String())
	bs.WriteString(" from ")
	bs.WriteString(table)

	bc := bytes.NewBufferString("select count(*) from ")
	bc.WriteString(table)

	if where != "" {
		bs.WriteString(" where ")
		bs.WriteString(where)

		bc.WriteString(" where ")
		bc.WriteString(where)
	}

	c := bc.String()
	log.Debugf("sql count:%v", c)

	if sort != "" {
		bs.WriteString(" order by ")
		bs.WriteString(sort)
		if order != "" {
			bs.WriteString(" ")
			bs.WriteString(order)
		}
	}

	if count > 0 {
		bs.WriteString(fmt.Sprintf(" limit %d,%d", offset, count))
	}

	sql := bs.String()
	log.Debugf("sql:%v", sql)

	db, err := mdb.GetConnection()
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer rows.Close()

	for rows.Next() {
		var refs []interface{}
		obj := reflect.New(fs)

		for i := 0; i < obj.Elem().NumField(); i++ {
			refs = append(refs, obj.Elem().Field(i).Addr().Interface())
		}

		if err := rows.Scan(refs...); err != nil {
			return 0, errors.Trace(err)
		}
		rv = reflect.Append(rv, obj.Elem())
	}

	reflect.ValueOf(result).Elem().Set(reflect.ValueOf(rv.Interface()))

	// select count
	row := db.QueryRow(c)
	row.Scan(&count)

	log.Debugf("result total:%d:%v", count, result)
	return count, nil
}

func updateProject(id int64, name, user, email, path, comments string) error {
	sql := "update project set name=?, user=?, email=?, path=?, comments=?, mtime=now() where id=?"
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()

	_, err = db.Exec(sql, name, user, email, path, comments, id)
	return errors.Trace(err)
}

func queryInterfaceInfo(id int64) (meta.Interface, error) {
	sql := "select name, user, email, state, method, level, path, backend, comments, ctime, mtime from interface where id=?"

	db, err := mdb.GetConnection()
	if err != nil {
		return meta.Interface{}, errors.Trace(err)
	}
	defer db.Close()
	rows, err := db.Query(sql, id)
	if err != nil {
		return meta.Interface{}, errors.Trace(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return meta.Interface{}, fmt.Errorf("interface id:%d not found", id)
	}

	i := meta.Interface{}
	if err = rows.Scan(&i.Name, &i.User, &i.Email, &i.State, &i.Method, &i.Level, &i.Path, &i.Backend, &i.Comments, &i.Ctime, &i.Mtime); err != nil {
		return meta.Interface{}, errors.Trace(err)
	}

	return i, nil
}

func deployInterface(id int64) error {
	sql := "update interface set state=1 where id=?"
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()
	_, err = db.Exec(sql, id)
	return errors.Trace(err)
}

func updateInterface(id int64, method, level int, name, path, backend, comments, user, email string) error {
	sql := "update interface set name=?, method=?,level=?, path=?, backend=?, comments=?, mtime=now(), user=?, email=? where id=?"
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()
	_, err = db.Exec(sql, name, method, level, path, backend, comments, user, email, id)
	return errors.Trace(err)
}

func updateVariable(id int64, postion int, name string, isNumber, isRequired int, example, comments string) error {
	sql := "update variable set postion=?, name =?, is_number=?, is_required=?, example=?, comments=?, mtime=now() where id=?"
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()
	_, err = db.Exec(sql, postion, name, isNumber, isRequired, example, comments, id)
	return errors.Trace(err)
}

func getApp(id int64) (meta.Application, error) {
	p := meta.Application{}
	sql := "select name, token, comments, ctime, mtime from application where id=?"

	db, err := mdb.GetConnection()
	if err != nil {
		return p, errors.Trace(err)
	}
	defer db.Close()
	rows, err := db.Query(sql, id)
	if err != nil {
		return p, errors.Trace(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return p, fmt.Errorf("app %d not found", id)
	}
	if err = rows.Scan(&p.Name, &p.Token, &p.Comments, &p.Ctime, &p.Mtime); err != nil {
		return p, errors.Trace(err)
	}
	p.ID = id

	return p, nil
}

func add(table string, data interface{}) (int64, error) {
	rt := reflect.TypeOf(data)
	rv := reflect.ValueOf(data)

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rv = rv.Elem()
	}

	if rt.NumField() == 0 {
		return 0, fmt.Errorf("data not found field")
	}

	bs := bytes.NewBufferString("insert into ")
	bs.WriteString(table)
	bs.WriteString(" (")

	for i := 0; i < rt.NumField(); i++ {
		name := rt.Field(i).Tag.Get("db")
		if name == "" {
			name = rt.Field(i).Name
		}
		bs.WriteString(name)
		bs.WriteString(", ")
	}
	bs.Truncate(bs.Len() - 2)

	bs.WriteString(") values (")
	for i := 0; i < rt.NumField(); i++ {
		switch rt.Field(i).Type.Kind() {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			bs.WriteString(fmt.Sprintf("%d, ", rv.Field(i).Int()))
		case reflect.Bool:
			if rv.Field(i).Bool() {
				bs.WriteString("1, ")
			} else {
				bs.WriteString("0, ")
			}
		case reflect.String:
			if rv.Field(i).String() == "" {
				bs.WriteString(rt.Field(i).Tag.Get("db_default") + ", ")
			} else {
				bs.WriteString("'" + rv.Field(i).String() + "', ")
			}
		}
	}
	bs.Truncate(bs.Len() - 2)
	bs.WriteString(")")

	sql := bs.String()
	log.Debugf("sql:%v", sql)
	db, err := mdb.GetConnection()
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer db.Close()
	r, err := db.Exec(sql)
	if err != nil {
		return 0, errors.Trace(err)
	}
	return r.LastInsertId()
}

func updateApp(where, name, user, email, comments string) error {
	sql := "update application set name=?, user=?, email=?, comments=?, mtime=now() where " + where
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()

	_, err = db.Exec(sql, name, user, email, comments)
	return errors.Trace(err)
}

func updateAppToken(id int64, token string) error {
	sql := "update application set token=?, mtime=now() where id=?"
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()

	_, err = db.Exec(sql, token, id)
	return errors.Trace(err)
}

func del(table string, id int64) error {
	sql := fmt.Sprintf("delete from %s where id=%d", table, id)
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()

	_, err = db.Exec(sql)
	return errors.Trace(err)
}

func updateRelation(id, iid, aid int64) error {
	sql := "update relation set interface_id=?, application_id=?, mtime=now() where id=?"
	db, err := mdb.GetConnection()
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()

	_, err = db.Exec(sql, iid, aid, id)
	return errors.Trace(err)
}

func getInterfaceState(id int64) (int, error) {
	var p int

	sql := "select state from interface where id=?"

	db, err := mdb.GetConnection()
	if err != nil {
		return p, errors.Trace(err)
	}
	defer db.Close()
	rows, err := db.Query(sql, id)
	if err != nil {
		return p, errors.Trace(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return p, fmt.Errorf("app %d not found", id)
	}
	if err = rows.Scan(&p); err != nil {
		return p, errors.Trace(err)
	}
	return p, nil
}

func getResourceID(table string, id int64) (int64, error) {
	sql := fmt.Sprintf("select resource_id from %s where id=%d", table, id)

	db, err := mdb.GetConnection()
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return 0, errors.Trace(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, fmt.Errorf("project %d not found", id)
	}

	var p int64
	if err = rows.Scan(&p); err != nil {
		return p, errors.Trace(err)
	}
	return p, nil
}

func selectStats(id int64) ([]statsSum, error) {
	sql := "SELECT DATE_FORMAT(ctime,'%Y/%m/%d %H:%i') , SUM(cnt), ROUND(SUM(cost) / sum(cnt)) FROM stats "
	if id > 0 {
		sql += fmt.Sprintf(" where stats.iface_id = %d ", id)
	}

	sql += "GROUP BY DATE_FORMAT(ctime,'%Y/%m/%d %H:%i') order by ctime desc limit 60;"

	db, err := mdb.GetConnection()
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	sss := []statsSum{}

	for rows.Next() {
		var ss statsSum
		if err = rows.Scan(&ss.Date, &ss.Sum, &ss.Avg); err != nil {
			return nil, errors.Trace(err)
		}
		sss = append(sss, ss)
	}

	return sss, nil
}

func selectTopIface() ([]statsTopIface, error) {
	sql := "SELECT i.id, p.name,i.name,i.user,sum(cnt) from stats as s,interface as i, project as p  where s.iface_id = i.id and  i.project_id = p.id and s.ctime > CURDATE()-interval 1 day GROUP BY iface_id ORDER BY sum(cnt) desc limit 10"
	db, err := mdb.GetConnection()
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer db.Close()

	rows, err := db.Query(sql)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	tis := []statsTopIface{}

	for rows.Next() {
		var ti statsTopIface
		if err = rows.Scan(&ti.ID, &ti.ProjectName, &ti.InterfaceName, &ti.User, &ti.Value); err != nil {
			return nil, errors.Trace(err)
		}
		tis = append(tis, ti)
	}

	return tis, nil
}

func selectTopApp(ifaceID int64) ([]statsTopApp, error) {
	sql := "SELECT a.id, a.name,a.user, i.id, i.name,i.user,p.id, p.name, sum(cnt) from stats as s,interface as i, application as a, project as p where "
	if ifaceID > 0 {
		sql += fmt.Sprintf("s.iface_id=%d and s.iface_id = i.id and  i.project_id = p.id and a.id = s.app_id and s.ctime > CURDATE()-interval 1 day GROUP BY iface_id ORDER BY sum(cnt) desc", ifaceID)
	}
	db, err := mdb.GetConnection()
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer db.Close()
	log.Debugf("sql:%v", sql)

	rows, err := db.Query(sql)
	if err != nil {
		return nil, errors.Trace(err)
	}
	defer rows.Close()

	tas := []statsTopApp{}

	for rows.Next() {
		var ta statsTopApp
		if err = rows.Scan(&ta.AppID, &ta.AppName, &ta.AppUser, &ta.InterfaceID, &ta.InterfaceName, &ta.InterfaceUser, &ta.ProjectID, &ta.ProjectName, &ta.Value); err != nil {
			return nil, errors.Trace(err)
		}
		tas = append(tas, ta)
	}

	return tas, nil
}
