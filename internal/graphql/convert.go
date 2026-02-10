package graphql

import (
	"afterzin/api/internal/graphql/model"
	"afterzin/api/internal/repository"
	"database/sql"
	"time"
)

func userRowToModel(u *repository.UserRow) *model.User {
	if u == nil {
		return nil
	}
	var photoURL *string
	if u.PhotoURL.Valid {
		photoURL = &u.PhotoURL.String
	}
	return &model.User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Cpf:       u.CPF,
		BirthDate: u.BirthDate,
		PhotoURL:  photoURL,
		Role:      model.UserRole(u.Role),
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func eventRowToModel(e *repository.EventRow, db *sql.DB) (*model.Event, error) {
	if e == nil {
		return nil, nil
	}
	var addr *string
	if e.Address.Valid {
		addr = &e.Address.String
	}
	feat := e.Featured == 1
	ev := &model.Event{
		ID:          e.ID,
		Title:       e.Title,
		Description: e.Description,
		Category:    e.Category,
		CoverImage:  e.CoverImage,
		Location:    e.Location,
		Address:     addr,
		Status:      model.EventStatus(e.Status),
		Featured:    &feat,
		Dates:       nil,
		Producer:    nil,
	}
	dateIDs, err := repository.EventDateIDsByEvent(db, e.ID)
	if err != nil {
		return nil, err
	}
	ev.Dates = make([]*model.EventDate, 0, len(dateIDs))
	for _, did := range dateIDs {
		ed, err := eventDateToModel(db, did)
		if err != nil {
			return nil, err
		}
		if ed != nil {
			ev.Dates = append(ev.Dates, ed)
		}
	}
	prodID := e.ProducerID
	prod, err := repository.ProducerByID(db, prodID)
	if err != nil {
		return nil, err
	}
	if prod != nil {
		owner, _ := repository.UserByID(db, prod.UserID)
		ev.Producer = &model.Producer{
			ID:          prod.ID,
			User:        userRowToModel(owner),
			CompanyName: nil,
			Approved:    prod.Approved == 1,
		}
		if prod.CompanyName.Valid {
			ev.Producer.CompanyName = &prod.CompanyName.String
		}
	}
	return ev, nil
}

func eventDateToModel(db *sql.DB, dateID string) (*model.EventDate, error) {
	d, err := repository.EventDateByID(db, dateID)
	if err != nil || d == nil {
		return nil, err
	}
	var st, et *string
	if d.StartTime.Valid {
		st = &d.StartTime.String
	}
	if d.EndTime.Valid {
		et = &d.EndTime.String
	}
	ed := &model.EventDate{
		ID:        d.ID,
		EventID:   d.EventID,
		Date:      d.Date,
		StartTime: st,
		EndTime:   et,
		Lots:      nil,
	}
	lotIDs, err := repository.LotIDsByEventDate(db, d.ID)
	if err != nil {
		return nil, err
	}
	ed.Lots = make([]*model.Lot, 0, len(lotIDs))
	for _, lid := range lotIDs {
		lot, err := lotToModel(db, lid)
		if err != nil {
			return nil, err
		}
		if lot != nil {
			ed.Lots = append(ed.Lots, lot)
		}
	}
	return ed, nil
}

func lotToModel(db *sql.DB, lotID string) (*model.Lot, error) {
	l, err := repository.LotByID(db, lotID)
	if err != nil || l == nil {
		return nil, err
	}
	lot := &model.Lot{
		ID:                l.ID,
		Name:              l.Name,
		StartsAt:          l.StartsAt,
		EndsAt:            l.EndsAt,
		TotalQuantity:     l.TotalQuantity,
		AvailableQuantity: l.AvailableQuantity,
		Active:            l.Active == 1,
		TicketTypes:       nil,
	}
	ttIDs, err := repository.TicketTypeIDsByLot(db, l.ID)
	if err != nil {
		return nil, err
	}
	lot.TicketTypes = make([]*model.TicketType, 0, len(ttIDs))
	for _, ttid := range ttIDs {
		tt, err := repository.TicketTypeByID(db, ttid)
		if err != nil || tt == nil {
			continue
		}
		var desc *string
		if tt.Description.Valid {
			desc = &tt.Description.String
		}
		lot.TicketTypes = append(lot.TicketTypes, &model.TicketType{
			ID:           tt.ID,
			Name:         tt.Name,
			Description:  desc,
			Price:        tt.Price,
			Audience:     model.AudienceType(tt.Audience),
			MaxQuantity:  tt.MaxQuantity,
			SoldQuantity: tt.SoldQuantity,
		})
	}
	return lot, nil
}

func ticketRowToModel(db *sql.DB, t *repository.TicketRow) (*model.Ticket, error) {
	if t == nil {
		return nil, nil
	}
	evRow, _ := repository.EventByID(db, t.EventID)
	ev, _ := eventRowToModel(evRow, db)
	ed, _ := eventDateToModel(db, t.EventDateID)
	tt, _ := repository.TicketTypeByID(db, t.TicketTypeID)
	var ttModel *model.TicketType
	if tt != nil {
		var desc *string
		if tt.Description.Valid {
			desc = &tt.Description.String
		}
		ttModel = &model.TicketType{
			ID:           tt.ID,
			Name:         tt.Name,
			Description:  desc,
			Price:        tt.Price,
			Audience:     model.AudienceType(tt.Audience),
			MaxQuantity:  tt.MaxQuantity,
			SoldQuantity: tt.SoldQuantity,
		}
	}
	owner, _ := repository.UserByID(db, t.UserID)
	ticket := &model.Ticket{
		ID:         t.ID,
		Code:       t.Code,
		QRCode:     t.QRCode,
		Event:      ev,
		EventDate:  ed,
		TicketType: ttModel,
		Owner:      userRowToModel(owner),
		Used:       t.Used == 1,
		CreatedAt:  t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if t.UsedAt.Valid && t.UsedAt.String != "" {
		usedAt := parseDateTimeToRFC3339(t.UsedAt.String)
		ticket.UsedAt = &usedAt
	}
	return ticket, nil
}

func parseDateTimeToRFC3339(s string) string {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t.UTC().Format(time.RFC3339)
}
