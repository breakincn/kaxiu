package handlers

import (
	"kabao/config"
	"kabao/models"
)

type usageCardSnapshot struct {
	CardNo      string
	CardType    string
	TotalTimes  int
	UsedTimes   int
	RemainTimes int
}

func intSnapshotPtr(v int) *int {
	value := v
	return &value
}

func applyUsageCardSnapshotFromCard(usage *models.Usage, card models.Card) {
	if usage == nil {
		return
	}
	usage.CardNoSnapshot = card.CardNo
	usage.CardTypeSnapshot = card.CardType
	usage.CardTotalTimesSnapshot = intSnapshotPtr(card.TotalTimes)
	usage.CardUsedTimesSnapshot = intSnapshotPtr(card.UsedTimes)
	usage.CardRemainTimesSnapshot = intSnapshotPtr(card.RemainTimes)
}

func usageHasCardSnapshot(usage models.Usage) bool {
	return usage.CardTotalTimesSnapshot != nil &&
		usage.CardUsedTimesSnapshot != nil &&
		usage.CardRemainTimesSnapshot != nil
}

func snapshotFromUsage(usage models.Usage, fallbackCard models.Card) usageCardSnapshot {
	snapshot := usageCardSnapshot{
		CardNo:      usage.CardNoSnapshot,
		CardType:    usage.CardTypeSnapshot,
		TotalTimes:  fallbackCard.TotalTimes,
		UsedTimes:   fallbackCard.UsedTimes,
		RemainTimes: fallbackCard.RemainTimes,
	}
	if snapshot.CardNo == "" {
		snapshot.CardNo = fallbackCard.CardNo
	}
	if snapshot.CardType == "" {
		snapshot.CardType = fallbackCard.CardType
	}
	if usage.CardTotalTimesSnapshot != nil {
		snapshot.TotalTimes = *usage.CardTotalTimesSnapshot
	}
	if usage.CardUsedTimesSnapshot != nil {
		snapshot.UsedTimes = *usage.CardUsedTimesSnapshot
	}
	if usage.CardRemainTimesSnapshot != nil {
		snapshot.RemainTimes = *usage.CardRemainTimesSnapshot
	}
	return snapshot
}

func applyCardSnapshotToUsage(usage *models.Usage, snapshot usageCardSnapshot) {
	if usage == nil {
		return
	}
	usage.CardNoSnapshot = snapshot.CardNo
	usage.CardTypeSnapshot = snapshot.CardType
	usage.CardTotalTimesSnapshot = intSnapshotPtr(snapshot.TotalTimes)
	usage.CardUsedTimesSnapshot = intSnapshotPtr(snapshot.UsedTimes)
	usage.CardRemainTimesSnapshot = intSnapshotPtr(snapshot.RemainTimes)
	usage.Card.CardNo = snapshot.CardNo
	usage.Card.CardType = snapshot.CardType
	usage.Card.TotalTimes = snapshot.TotalTimes
	usage.Card.UsedTimes = snapshot.UsedTimes
	usage.Card.RemainTimes = snapshot.RemainTimes
}

func enrichUsagesWithCardSnapshots(usages []models.Usage) {
	if len(usages) == 0 {
		return
	}

	cardIDSet := make(map[uint]bool)
	for i := range usages {
		if usages[i].CardID > 0 {
			cardIDSet[usages[i].CardID] = true
		}
	}
	if len(cardIDSet) == 0 {
		return
	}

	cardIDs := make([]uint, 0, len(cardIDSet))
	for id := range cardIDSet {
		cardIDs = append(cardIDs, id)
	}

	var cards []models.Card
	if err := config.DB.Where("id IN ?", cardIDs).Find(&cards).Error; err != nil {
		return
	}
	cardByID := make(map[uint]models.Card, len(cards))
	for _, card := range cards {
		cardByID[card.ID] = card
	}

	var allUsages []models.Usage
	if err := config.DB.
		Model(&models.Usage{}).
		Select("id, card_id, used_times, card_no_snapshot, card_type_snapshot, card_total_times_snapshot, card_used_times_snapshot, card_remain_times_snapshot").
		Where("card_id IN ? AND status <> ?", cardIDs, "failed").
		Order("card_id asc, COALESCE(used_at, created_at) desc, id desc").
		Find(&allUsages).Error; err != nil {
		return
	}

	nextSnapshotByCardID := make(map[uint]usageCardSnapshot, len(cardByID))
	for id, card := range cardByID {
		nextSnapshotByCardID[id] = usageCardSnapshot{
			CardNo:      card.CardNo,
			CardType:    card.CardType,
			TotalTimes:  card.TotalTimes,
			UsedTimes:   card.UsedTimes,
			RemainTimes: card.RemainTimes,
		}
	}

	snapshotByUsageID := make(map[uint]usageCardSnapshot, len(allUsages))
	for _, usage := range allUsages {
		currentSnapshot, ok := nextSnapshotByCardID[usage.CardID]
		if !ok {
			continue
		}
		if usageHasCardSnapshot(usage) {
			snapshotByUsageID[usage.ID] = snapshotFromUsage(usage, cardByID[usage.CardID])
		} else {
			snapshotByUsageID[usage.ID] = currentSnapshot
		}

		usedTimes := usage.UsedTimes
		if usedTimes < 0 {
			usedTimes = 0
		}
		currentSnapshot.UsedTimes -= usedTimes
		if currentSnapshot.UsedTimes < 0 {
			currentSnapshot.UsedTimes = 0
		}
		currentSnapshot.RemainTimes += usedTimes
		nextSnapshotByCardID[usage.CardID] = currentSnapshot
	}

	for i := range usages {
		snapshot, ok := snapshotByUsageID[usages[i].ID]
		if !ok {
			continue
		}
		applyCardSnapshotToUsage(&usages[i], snapshot)
	}
}

func enrichServiceSessionsWithCardSnapshots(sessions []models.ServiceSession) {
	if len(sessions) == 0 {
		return
	}

	usages := make([]models.Usage, 0, len(sessions))
	for i := range sessions {
		session := &sessions[i]
		if session.InitialUsage != nil && session.InitialUsage.ID > 0 {
			usage := *session.InitialUsage
			if usage.CardID == 0 {
				usage.CardID = session.CardID
			}
			usages = append(usages, usage)
			continue
		}
		if session.InitialUsageID > 0 {
			usages = append(usages, models.Usage{ID: session.InitialUsageID, CardID: session.CardID})
		}
	}
	if len(usages) == 0 {
		return
	}

	enrichUsagesWithCardSnapshots(usages)
	usageByID := make(map[uint]models.Usage, len(usages))
	for _, usage := range usages {
		usageByID[usage.ID] = usage
	}

	for i := range sessions {
		session := &sessions[i]
		usage, ok := usageByID[session.InitialUsageID]
		if !ok {
			continue
		}
		if session.InitialUsage == nil {
			session.InitialUsage = &usage
		} else {
			session.InitialUsage.CardNoSnapshot = usage.CardNoSnapshot
			session.InitialUsage.CardTypeSnapshot = usage.CardTypeSnapshot
			session.InitialUsage.CardTotalTimesSnapshot = usage.CardTotalTimesSnapshot
			session.InitialUsage.CardUsedTimesSnapshot = usage.CardUsedTimesSnapshot
			session.InitialUsage.CardRemainTimesSnapshot = usage.CardRemainTimesSnapshot
		}
		if session.Card != nil {
			cardSnapshot := *session.Card
			cardSnapshot.CardNo = usage.CardNoSnapshot
			cardSnapshot.CardType = usage.CardTypeSnapshot
			if usage.CardTotalTimesSnapshot != nil {
				cardSnapshot.TotalTimes = *usage.CardTotalTimesSnapshot
			}
			if usage.CardUsedTimesSnapshot != nil {
				cardSnapshot.UsedTimes = *usage.CardUsedTimesSnapshot
			}
			if usage.CardRemainTimesSnapshot != nil {
				cardSnapshot.RemainTimes = *usage.CardRemainTimesSnapshot
			}
			session.Card = &cardSnapshot
		}
	}
}
