package httpapi

import (
	"net/http"

	"github.com/kadebhug/seatd_v2/internal/domain/identity"
)

type ownerSetupRequest struct {
	Floor          ownerOnboardingFloorRequest       `json:"floor"`
	ServicePeriods []ownerOnboardingServicePeriodDTO `json:"servicePeriods"`
}

type ownerSetupResponse struct {
	Organisation   organisationDTO    `json:"organisation"`
	Location       locationDTO        `json:"location"`
	Floor          *floorDTO          `json:"floor,omitempty"`
	Zones          []zoneDTO          `json:"zones"`
	Tables         []tableDTO         `json:"tables"`
	ServicePeriods []servicePeriodDTO `json:"servicePeriods"`
}

func (api *API) setupOwner(w http.ResponseWriter, r *http.Request) {
	secret, ok := bearerCredential(w, r)
	if !ok {
		return
	}
	session, err := api.ident.ValidateWebSession(r.Context(), secret)
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	var req ownerSetupRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	result, err := api.ident.OwnerSetup(r.Context(), identity.OwnerSetupParams{
		UserProfileID:  session.User.ID,
		Floor:          ownerSetupFloorFromRequest(req.Floor),
		ServicePeriods: ownerSetupServicePeriodsFromRequest(req.ServicePeriods),
	})
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ownerSetupResponseFromDomain(result))
}

func ownerSetupFloorFromRequest(req ownerOnboardingFloorRequest) identity.OnboardingFloor {
	zones := make([]identity.OnboardingZone, 0, len(req.Zones))
	for _, zone := range req.Zones {
		zones = append(zones, identity.OnboardingZone{
			Name:      zone.Name,
			SortOrder: zone.SortOrder,
		})
	}
	tables := make([]identity.OnboardingTable, 0, len(req.Tables))
	for _, table := range req.Tables {
		tables = append(tables, identity.OnboardingTable{
			Label:         table.Label,
			CapacityLabel: table.CapacityLabel,
			Shape:         table.Shape,
			Geometry:      table.Geometry,
			ZoneName:      table.ZoneName,
		})
	}
	return identity.OnboardingFloor{
		Name:      req.Name,
		Slug:      req.Slug,
		Canvas:    req.Canvas,
		Zones:     zones,
		Tables:    tables,
		SortOrder: req.SortOrder,
	}
}

func ownerSetupServicePeriodsFromRequest(req []ownerOnboardingServicePeriodDTO) []identity.OnboardingServicePeriod {
	periods := make([]identity.OnboardingServicePeriod, 0, len(req))
	for _, period := range req {
		periods = append(periods, identity.OnboardingServicePeriod{
			Name:       period.Name,
			DaysOfWeek: period.DaysOfWeek,
			StartTime:  period.StartTime,
			EndTime:    period.EndTime,
		})
	}
	return periods
}

func ownerSetupResponseFromDomain(result identity.OwnerSetupResult) ownerSetupResponse {
	var floor *floorDTO
	if result.Floor != nil {
		dto := floorFromDB(*result.Floor)
		floor = &dto
	}
	zones := make([]zoneDTO, 0, len(result.Zones))
	for _, zone := range result.Zones {
		zones = append(zones, zoneFromDB(zone))
	}
	tables := make([]tableDTO, 0, len(result.Tables))
	for _, table := range result.Tables {
		tables = append(tables, tableFromDB(table))
	}
	periods := make([]servicePeriodDTO, 0, len(result.ServicePeriods))
	for _, period := range result.ServicePeriods {
		periods = append(periods, servicePeriodFromDB(period))
	}
	return ownerSetupResponse{
		Organisation:   organisationFromDB(result.Organisation),
		Location:       locationFromDB(result.Location),
		Floor:          floor,
		Zones:          zones,
		Tables:         tables,
		ServicePeriods: periods,
	}
}
