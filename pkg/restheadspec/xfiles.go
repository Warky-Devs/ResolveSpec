package restheadspec

import (
	"encoding/json"
	"reflect"
)

type XFiles struct {
	TableName   string      `json:"tablename"`
	Schema      string      `json:"schema"`
	PrimaryKey  string      `json:"primarykey"`
	ForeignKey  string      `json:"foreignkey"`
	RelatedKey  string      `json:"relatedkey"`
	Sort        []string    `json:"sort"`
	Prefix      string      `json:"prefix"`
	Editable    bool        `json:"editable"`
	Recursive   bool        `json:"recursive"`
	Expand      bool        `json:"expand"`
	Rownumber   bool        `json:"rownumber"`
	Skipcount   bool        `json:"skipcount"`
	Offset      json.Number `json:"offset"`
	Limit       json.Number `json:"limit"`
	Columns     []string    `json:"columns"`
	OmitColumns []string    `json:"omit_columns"`
	CQLColumns  []string    `json:"cql_columns"`

	SqlJoins     []string     `json:"sql_joins"`
	SqlOr        []string     `json:"sql_or"`
	SqlAnd       []string     `json:"sql_and"`
	ParentTables []*XFiles    `json:"parenttables"`
	ChildTables  []*XFiles    `json:"childtables"`
	ModelType    reflect.Type `json:"-"`
	ParentEntity *XFiles      `json:"-"`
	Level        uint         `json:"-"`
	Errors       []error      `json:"-"`
	FilterFields []struct {
		Field    string `json:"field"`
		Value    string `json:"value"`
		Operator string `json:"operator"`
	} `json:"filter_fields"`
	CursorForward  string `json:"cursor_forward"`
	CursorBackward string `json:"cursor_backward"`
}

// func (m *XFiles) SetParent() {
// 	if m.ChildTables != nil {
// 		for _, child := range m.ChildTables {
// 			if child.ParentEntity != nil {
// 				continue
// 			}
// 			child.ParentEntity = m
// 			child.Level = m.Level + 1000
// 			child.SetParent()
// 		}
// 	}
// 	if m.ParentTables != nil {
// 		for _, pt := range m.ParentTables {
// 			if pt.ParentEntity != nil {
// 				continue
// 			}
// 			pt.ParentEntity = m
// 			pt.Level = m.Level + 1
// 			pt.SetParent()
// 		}
// 	}
// }

// func (m *XFiles) GetParentRelations() []reflection.GormRelationType {
// 	if m.ParentEntity == nil {
// 		return nil
// 	}

// 	foundRelations := make(GormRelationTypeList, 0)
// 	rels := reflection.GetValidModelRelationTypes(m.ParentEntity.ModelType, false)

// 	if m.ParentEntity.ModelType == nil {
// 		return nil
// 	}

// 	for _, rel := range rels {
// 		// if len(foundRelations) > 0 {
// 		// 	break
// 		// }
// 		if rel.FieldName != "" && rel.AssociationTable.Name() == m.ModelType.Name() {

// 			if rel.AssociationKey != "" && m.RelatedKey != "" && strings.EqualFold(rel.AssociationKey, m.RelatedKey) {
// 				foundRelations = append(foundRelations, rel)
// 			} else if rel.AssociationKey != "" && m.ForeignKey != "" && strings.EqualFold(rel.AssociationKey, m.ForeignKey) {
// 				foundRelations = append(foundRelations, rel)
// 			} else if rel.ForeignKey != "" && m.ForeignKey != "" && strings.EqualFold(rel.ForeignKey, m.ForeignKey) {
// 				foundRelations = append(foundRelations, rel)
// 			} else if rel.ForeignKey != "" && m.RelatedKey != "" && strings.EqualFold(rel.ForeignKey, m.RelatedKey) {
// 				foundRelations = append(foundRelations, rel)
// 			} else if rel.ForeignKey != "" && m.ForeignKey == "" && m.RelatedKey == "" {
// 				foundRelations = append(foundRelations, rel)
// 			}
// 		}

// 		//idName := fmt.Sprintf("%s_to_%s_%s=%s_m%v", rel.TableName, rel.AssociationTableName, rel.ForeignKey, rel.AssociationKey, rel.OneToMany)
// 	}

// 	sort.Sort(foundRelations)
// 	finalList := make(GormRelationTypeList, 0)
// 	dups := make(map[string]bool)
// 	for _, rel := range foundRelations {
// 		idName := fmt.Sprintf("%s_to_%s_%s_%s=%s_m%v", rel.TableName, rel.AssociationTableName, rel.FieldName, rel.ForeignKey, rel.AssociationKey, rel.OneToMany)
// 		if dups[idName] {
// 			continue
// 		}
// 		finalList = append(finalList, rel)
// 		dups[idName] = true
// 	}

// 	//fmt.Printf("GetParentRelations %s: %+v  %d=%d\n", m.TableName, dups, len(finalList), len(foundRelations))

// 	return finalList
// }

// func (m *XFiles) GetUpdatableTableNames() []string {
// 	foundTables := make([]string, 0)
// 	if m.Editable {
// 		foundTables = append(foundTables, m.TableName)
// 	}
// 	if m.ParentTables != nil {
// 		for _, pt := range m.ParentTables {
// 			list := pt.GetUpdatableTableNames()
// 			if list != nil {
// 				foundTables = append(foundTables, list...)
// 			}
// 		}
// 	}
// 	if m.ChildTables != nil {
// 		for _, ct := range m.ChildTables {
// 			list := ct.GetUpdatableTableNames()
// 			if list != nil {
// 				foundTables = append(foundTables, list...)

// 			}
// 		}
// 	}

// 	return foundTables
// }

// func (m *XFiles) preload(db *gorm.DB, pPath string, pCnt uint) (*gorm.DB, error) {

// 	path := pPath
// 	_, colval := JSONSyntaxToSQLIn(path, m.ModelType, "preload")
// 	if colval != "" {
// 		path = colval
// 	}

// 	if path == "" {
// 		return db, fmt.Errorf("invalid preload path %s", path)
// 	}

// 	sortList := ""
// 	if m.Sort != nil {
// 		for _, sort := range m.Sort {
// 			descSort := false
// 			if strings.HasPrefix(sort, "-") || strings.Contains(strings.ToLower(sort), " desc") {
// 				descSort = true
// 			}
// 			sort = strings.TrimPrefix(strings.TrimPrefix(sort, "+"), "-")
// 			sort = strings.ReplaceAll(strings.ReplaceAll(sort, " desc", ""), " asc", "")
// 			if descSort {
// 				sort = sort + " desc"
// 			}
// 			sortList = sort
// 		}
// 	}

// 	SrcColumns := reflection.GetModelSQLColumns(m.ModelType)
// 	Columns := make([]string, 0)

// 	for _, s := range SrcColumns {
// 		for _, v := range m.Columns {
// 			if strings.EqualFold(v, s) {
// 				Columns = append(Columns, v)
// 				break
// 			}
// 		}
// 	}

// 	if len(Columns) == 0 {
// 		Columns = SrcColumns
// 	}

// 	chain := db

// 	// //Do expand where we can
// 	// if m.Expand {
// 	// 	ops := func(subchain *gorm.DB) *gorm.DB {
// 	// 		subchain = subchain.Select(strings.Join(m.Columns, ","))

// 	// 		if m.Filter != "" {
// 	// 			subchain = subchain.Where(m.Filter)
// 	// 		}
// 	// 		return subchain
// 	// 	}
// 	// 	chain = chain.Joins(path, ops(chain))
// 	// }

// 	//fmt.Printf("Preloading %s: %s   lvl:%d \n", m.TableName, path, m.Level)
// 	//Do preload
// 	chain = chain.Preload(path, func(db *gorm.DB) *gorm.DB {
// 		subchain := db

// 		if sortList != "" {
// 			subchain = subchain.Order(sortList)
// 		}

// 		for _, sql := range m.SqlAnd {
// 			fnType, colval := JSONSyntaxToSQL(sql, m.ModelType)
// 			if fnType == 0 {
// 				colval = ValidSQL(colval, "select")
// 			}
// 			subchain = subchain.Where(colval)
// 		}

// 		for _, sql := range m.SqlOr {
// 			fnType, colval := JSONSyntaxToSQL(sql, m.ModelType)
// 			if fnType == 0 {
// 				colval = ValidSQL(colval, "select")
// 			}
// 			subchain = subchain.Or(colval)
// 		}

// 		limitval, err := m.Limit.Int64()
// 		if err == nil && limitval > 0 {
// 			subchain = subchain.Limit(int(limitval))
// 		}

// 		for _, j := range m.SqlJoins {
// 			subchain = subchain.Joins(ValidSQL(j, "select"))
// 		}

// 		offsetval, err := m.Offset.Int64()
// 		if err == nil && offsetval > 0 {
// 			subchain = subchain.Offset(int(offsetval))
// 		}

// 		cols := make([]string, 0)

// 		for _, col := range Columns {
// 			canAdd := true
// 			for _, omit := range m.OmitColumns {
// 				if col == omit {
// 					canAdd = false
// 					break
// 				}
// 			}
// 			if canAdd {
// 				cols = append(cols, col)
// 			}
// 		}

// 		for i, col := range m.CQLColumns {
// 			cols = append(cols, fmt.Sprintf("(%s) as cql%d", col, i+1))
// 		}

// 		if len(cols) > 0 {

// 			colStr := strings.Join(cols, ",")
// 			subchain = subchain.Select(colStr)
// 		}

// 		if m.Recursive && pCnt < 5 {
// 			paths := strings.Split(path, ".")

// 			p := paths[0]
// 			if len(paths) > 1 {
// 				p = strings.Join(paths[1:], ".")
// 			}
// 			for i := uint(0); i < 3; i++ {
// 				inlineStr := strings.Repeat(p+".", int(i+1))
// 				inlineStr = strings.TrimRight(inlineStr, ".")

// 				fmt.Printf("Preloading Recursive (%d) %s: %s   lvl:%d \n", i, m.TableName, inlineStr, m.Level)
// 				subchain, err = m.preload(subchain, inlineStr, pCnt+i)
// 				if err != nil {
// 					cfg.LogError("Preload (%s,%d) error: %v", m.TableName, pCnt, err)
// 				} else {

// 					if m.ChildTables != nil {
// 						for _, child := range m.ChildTables {
// 							if child.ParentEntity == nil {
// 								continue
// 							}
// 							subchain, _ = child.ChainPreload(subchain, inlineStr, pCnt+i)

// 						}
// 					}
// 					if m.ParentTables != nil {
// 						for _, pt := range m.ParentTables {
// 							if pt.ParentEntity == nil {
// 								continue
// 							}
// 							subchain, _ = pt.ChainPreload(subchain, inlineStr, pCnt+i)

// 						}
// 					}

// 				}
// 			}

// 		}

// 		return subchain
// 	})

// 	return chain, nil

// }

// func (m *XFiles) ChainPreload(db *gorm.DB, pPath string, pCnt uint) (*gorm.DB, error) {
// 	var err error
// 	chain := db

// 	relations := m.GetParentRelations()
// 	if pCnt > 10000 {
// 		cfg.LogError("Preload Max size (%s,%s): %v", m.TableName, pPath, err)
// 		return chain, nil
// 	}

// 	hasPreloadError := false
// 	for _, rel := range relations {
// 		path := rel.FieldName
// 		if pPath != "" {
// 			path = fmt.Sprintf("%s.%s", pPath, rel.FieldName)
// 		}

// 		chain, err = m.preload(chain, path, pCnt)
// 		if err != nil {
// 			cfg.LogError("Preload Error (%s,%s): %v", m.TableName, path, err)
// 			hasPreloadError = true
// 			//return chain, err
// 		}

// 		//fmt.Printf("Preloading Rel %v: %s  @ %s lvl:%d \n", m.Recursive, path, m.TableName, m.Level)
// 		if !hasPreloadError && m.ChildTables != nil {
// 			for _, child := range m.ChildTables {
// 				if child.ParentEntity == nil {
// 					continue
// 				}
// 				chain, err = child.ChainPreload(chain, path, pCnt)
// 				if err != nil {
// 					return chain, err
// 				}
// 			}
// 		}
// 		if !hasPreloadError && m.ParentTables != nil {
// 			for _, pt := range m.ParentTables {
// 				if pt.ParentEntity == nil {
// 					continue
// 				}
// 				chain, err = pt.ChainPreload(chain, path, pCnt)
// 				if err != nil {
// 					return chain, err
// 				}
// 			}
// 		}
// 	}

// 	if len(relations) == 0 {
// 		if m.ChildTables != nil {
// 			for _, child := range m.ChildTables {
// 				if child.ParentEntity == nil {
// 					continue
// 				}
// 				chain, err = child.ChainPreload(chain, pPath, pCnt)
// 				if err != nil {
// 					return chain, err
// 				}
// 			}
// 		}
// 		if m.ParentTables != nil {
// 			for _, pt := range m.ParentTables {
// 				if pt.ParentEntity == nil {
// 					continue
// 				}
// 				chain, err = pt.ChainPreload(chain, pPath, pCnt)
// 				if err != nil {
// 					return chain, err
// 				}
// 			}
// 		}
// 	}

// 	return chain, nil
// }

// func (m *XFiles) Fill() {
// 	m.ModelType = models.GetModelType(m.Schema, m.TableName)

// 	if m.ModelType == nil {
// 		m.Errors = append(m.Errors, fmt.Errorf("ModelType not found for %s", m.TableName))
// 	}
// 	if m.Prefix == "" {
// 		m.Prefix = reflection.GetTablePrefixFromType(m.ModelType)
// 	}
// 	if m.PrimaryKey == "" {
// 		m.PrimaryKey = reflection.GetPKNameFromType(m.ModelType)
// 	}

// 	if m.Schema == "" {
// 		m.Schema = reflection.GetSchemaNameFromType(m.ModelType)
// 	}

// 	for _, t := range m.ParentTables {
// 		t.Fill()
// 	}

// 	for _, t := range m.ChildTables {
// 		t.Fill()
// 	}
// }

// type GormRelationTypeList []reflection.GormRelationType

// func (s GormRelationTypeList) Len() int      { return len(s) }
// func (s GormRelationTypeList) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// func (s GormRelationTypeList) Less(i, j int) bool {
// 	if strings.HasPrefix(strings.ToLower(s[j].FieldName),
// 		strings.ToLower(fmt.Sprintf("%s_%s_%s", s[i].AssociationSchema, s[i].AssociationTable, s[i].AssociationKey))) {
// 		return true
// 	}

// 	return s[i].FieldName < s[j].FieldName
// }
