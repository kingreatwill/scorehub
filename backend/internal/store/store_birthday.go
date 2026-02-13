package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateBirthdayContact(ctx context.Context, userID int64, in BirthdayContactInput) (BirthdayContact, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return BirthdayContact{}, ErrInvalidArgument
	}

	var contact BirthdayContact
	var solar sql.NullTime
	if in.SolarBirthday != nil {
		solar = sql.NullTime{Valid: true, Time: *in.SolarBirthday}
	}
	err := s.pool.QueryRow(ctx, `
INSERT INTO birthday_contacts
  (user_id, name, gender, phone, relation, note, avatar_url,
   solar_birthday, lunar_birthday, primary_type, primary_month, primary_day, primary_year, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7,
        $8, $9, $10, $11, $12, $13, NOW())
RETURNING id::text, user_id, name, gender, phone, relation, note, avatar_url,
          solar_birthday, lunar_birthday, primary_type, primary_month, primary_day, primary_year,
          created_at, updated_at
`, userID, name, in.Gender, in.Phone, in.Relation, in.Note, in.AvatarURL,
		solar, in.LunarBirthday, in.PrimaryType, in.PrimaryMonth, in.PrimaryDay, in.PrimaryYear).Scan(
		&contact.ID,
		&contact.UserID,
		&contact.Name,
		&contact.Gender,
		&contact.Phone,
		&contact.Relation,
		&contact.Note,
		&contact.AvatarURL,
		&solar,
		&contact.LunarBirthday,
		&contact.PrimaryType,
		&contact.PrimaryMonth,
		&contact.PrimaryDay,
		&contact.PrimaryYear,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)
	if err != nil {
		return BirthdayContact{}, err
	}
	if solar.Valid {
		t := solar.Time
		contact.SolarBirthday = &t
	}
	return contact, nil
}

func (s *Store) GetBirthdayContact(ctx context.Context, userID int64, id string) (BirthdayContact, error) {
	var contact BirthdayContact
	var solar sql.NullTime
	err := s.pool.QueryRow(ctx, `
SELECT id::text, user_id, name, gender, phone, relation, note, avatar_url,
       solar_birthday, lunar_birthday, primary_type, primary_month, primary_day, primary_year,
       created_at, updated_at
FROM birthday_contacts
WHERE id = $1::uuid AND user_id = $2
`, id, userID).Scan(
		&contact.ID,
		&contact.UserID,
		&contact.Name,
		&contact.Gender,
		&contact.Phone,
		&contact.Relation,
		&contact.Note,
		&contact.AvatarURL,
		&solar,
		&contact.LunarBirthday,
		&contact.PrimaryType,
		&contact.PrimaryMonth,
		&contact.PrimaryDay,
		&contact.PrimaryYear,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BirthdayContact{}, ErrNotFound
		}
		return BirthdayContact{}, err
	}
	if solar.Valid {
		t := solar.Time
		contact.SolarBirthday = &t
	}
	return contact, nil
}

func (s *Store) ListBirthdayContacts(ctx context.Context, userID int64) ([]BirthdayContactWithDays, error) {
	rows, err := s.pool.Query(ctx, `
WITH base AS (
  SELECT
    id::text,
    user_id,
    name,
    gender,
    phone,
    relation,
    note,
    avatar_url,
    solar_birthday,
    lunar_birthday,
    primary_type,
    primary_month,
    primary_day,
    primary_year,
    created_at,
    updated_at,
    GREATEST(1, LEAST(primary_month, 12))::int AS safe_month,
    GREATEST(1, LEAST(primary_day, 31))::int AS safe_day
  FROM birthday_contacts
  WHERE user_id = $1
),
calc AS (
  SELECT *,
    make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int, safe_month, 1) AS base_this_year,
    make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int + 1, safe_month, 1) AS base_next_year
  FROM base
),
days AS (
  SELECT *,
    LEAST(safe_day, EXTRACT(day FROM (base_this_year + interval '1 month - 1 day')))::int AS day_this_year,
    LEAST(safe_day, EXTRACT(day FROM (base_next_year + interval '1 month - 1 day')))::int AS day_next_year
  FROM calc
),
dates AS (
  SELECT *,
    make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int, safe_month, day_this_year) AS birthday_this_year,
    make_date(EXTRACT(YEAR FROM CURRENT_DATE)::int + 1, safe_month, day_next_year) AS birthday_next_year
  FROM days
)
SELECT
  id, user_id, name, gender, phone, relation, note, avatar_url,
  solar_birthday, lunar_birthday, primary_type, primary_month, primary_day, primary_year,
  created_at, updated_at,
  CASE WHEN birthday_this_year < CURRENT_DATE THEN birthday_next_year ELSE birthday_this_year END AS next_birthday,
  CASE WHEN birthday_this_year < CURRENT_DATE THEN (birthday_next_year - CURRENT_DATE)
       ELSE (birthday_this_year - CURRENT_DATE) END AS days_left
FROM dates
ORDER BY days_left ASC, name ASC
`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []BirthdayContactWithDays
	for rows.Next() {
		var item BirthdayContactWithDays
		var solar sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Name,
			&item.Gender,
			&item.Phone,
			&item.Relation,
			&item.Note,
			&item.AvatarURL,
			&solar,
			&item.LunarBirthday,
			&item.PrimaryType,
			&item.PrimaryMonth,
			&item.PrimaryDay,
			&item.PrimaryYear,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.NextBirthday,
			&item.DaysLeft,
		); err != nil {
			return nil, err
		}
		if solar.Valid {
			t := solar.Time
			item.SolarBirthday = &t
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) UpdateBirthdayContact(ctx context.Context, userID int64, id string, in BirthdayContactUpdate) (BirthdayContact, error) {
	hasUpdate := false
	if in.Name != nil || in.Gender != nil || in.Phone != nil || in.Relation != nil || in.Note != nil ||
		in.AvatarURL != nil || in.SolarBirthday != nil || in.SolarSetNull || in.LunarBirthday != nil ||
		in.PrimaryType != nil || in.PrimaryMonth != nil || in.PrimaryDay != nil || in.PrimaryYear != nil {
		hasUpdate = true
	}
	if !hasUpdate {
		return BirthdayContact{}, ErrInvalidArgument
	}

	var name sql.NullString
	if in.Name != nil {
		name = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Name)}
	}
	var gender sql.NullString
	if in.Gender != nil {
		gender = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Gender)}
	}
	var phone sql.NullString
	if in.Phone != nil {
		phone = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Phone)}
	}
	var relation sql.NullString
	if in.Relation != nil {
		relation = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Relation)}
	}
	var note sql.NullString
	if in.Note != nil {
		note = sql.NullString{Valid: true, String: strings.TrimSpace(*in.Note)}
	}
	var avatar sql.NullString
	if in.AvatarURL != nil {
		avatar = sql.NullString{Valid: true, String: strings.TrimSpace(*in.AvatarURL)}
	}
	var solar sql.NullTime
	if in.SolarBirthday != nil {
		solar = sql.NullTime{Valid: true, Time: *in.SolarBirthday}
	}
	var lunar sql.NullString
	if in.LunarBirthday != nil {
		lunar = sql.NullString{Valid: true, String: strings.TrimSpace(*in.LunarBirthday)}
	}
	var primaryType sql.NullString
	if in.PrimaryType != nil {
		primaryType = sql.NullString{Valid: true, String: strings.TrimSpace(*in.PrimaryType)}
	}
	var primaryMonth sql.NullInt64
	if in.PrimaryMonth != nil {
		primaryMonth = sql.NullInt64{Valid: true, Int64: int64(*in.PrimaryMonth)}
	}
	var primaryDay sql.NullInt64
	if in.PrimaryDay != nil {
		primaryDay = sql.NullInt64{Valid: true, Int64: int64(*in.PrimaryDay)}
	}
	var primaryYear sql.NullInt64
	if in.PrimaryYear != nil {
		primaryYear = sql.NullInt64{Valid: true, Int64: int64(*in.PrimaryYear)}
	}

	var contact BirthdayContact
	var solarOut sql.NullTime
	err := s.pool.QueryRow(ctx, `
UPDATE birthday_contacts
SET name = COALESCE($3, name),
    gender = COALESCE($4, gender),
    phone = COALESCE($5, phone),
    relation = COALESCE($6, relation),
    note = COALESCE($7, note),
    avatar_url = COALESCE($8, avatar_url),
    solar_birthday = CASE WHEN $9 THEN NULL ELSE COALESCE($10, solar_birthday) END,
    lunar_birthday = COALESCE($11, lunar_birthday),
    primary_type = COALESCE($12, primary_type),
    primary_month = COALESCE($13, primary_month),
    primary_day = COALESCE($14, primary_day),
    primary_year = COALESCE($15, primary_year),
    updated_at = NOW()
WHERE id = $1::uuid AND user_id = $2
RETURNING id::text, user_id, name, gender, phone, relation, note, avatar_url,
          solar_birthday, lunar_birthday, primary_type, primary_month, primary_day, primary_year,
          created_at, updated_at
`, id, userID,
		name, gender, phone, relation, note, avatar,
		in.SolarSetNull, solar, lunar, primaryType, primaryMonth, primaryDay, primaryYear,
	).Scan(
		&contact.ID,
		&contact.UserID,
		&contact.Name,
		&contact.Gender,
		&contact.Phone,
		&contact.Relation,
		&contact.Note,
		&contact.AvatarURL,
		&solarOut,
		&contact.LunarBirthday,
		&contact.PrimaryType,
		&contact.PrimaryMonth,
		&contact.PrimaryDay,
		&contact.PrimaryYear,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BirthdayContact{}, ErrNotFound
		}
		return BirthdayContact{}, err
	}
	if solarOut.Valid {
		t := solarOut.Time
		contact.SolarBirthday = &t
	}
	return contact, nil
}

func (s *Store) DeleteBirthdayContact(ctx context.Context, userID int64, id string) error {
	tag, err := s.pool.Exec(ctx, `
DELETE FROM birthday_contacts
WHERE id = $1::uuid AND user_id = $2
`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
