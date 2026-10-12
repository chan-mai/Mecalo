package httpapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"

	"github.com/chan-mai/mecalo/backend/internal/adapter/httpapi/gen"
	"github.com/chan-mai/mecalo/backend/internal/app"
	"github.com/chan-mai/mecalo/backend/internal/domain/body"
	"github.com/chan-mai/mecalo/backend/internal/domain/food"
	"github.com/chan-mai/mecalo/backend/internal/domain/meal"
	"github.com/chan-mai/mecalo/backend/internal/domain/nutrient"
	"github.com/chan-mai/mecalo/backend/internal/domain/user"
)

var _ gen.StrictServerInterface = (*server)(nil)

// 認証

func (s *server) AuthApiConfig(ctx context.Context, _ gen.AuthApiConfigRequestObject) (gen.AuthApiConfigResponseObject, error) {
	setup, err := s.app.Accounts.NeedsSetup(ctx)
	if err != nil {
		return nil, err
	}
	auth := s.opt.Auth()
	res := gen.AuthApiConfig200JSONResponse{
		SetupRequired:    setup && auth.Local,
		RegistrationOpen: s.app.Accounts.RegistrationOpen() && auth.Local,
		LocalEnabled:     auth.Local,
		OidcEnabled:      auth.OIDC != nil,
	}
	if auth.OIDC != nil {
		name := auth.OIDC.Name()
		res.OidcName = &name
	}
	return res, nil
}

func signUpInput(b gen.SignUpRequest) app.SignUpInput {
	return app.SignUpInput{LoginName: b.LoginName, DisplayName: str(b.DisplayName), Password: b.Password}
}

func (s *server) AuthApiSetup(ctx context.Context, req gen.AuthApiSetupRequestObject) (gen.AuthApiSetupResponseObject, error) {
	if !s.opt.Auth().Local {
		return nil, app.ErrDisabled
	}
	b := req.Body
	sess, err := s.app.Accounts.Setup(ctx, b.SetupToken, signUpInput(gen.SignUpRequest{LoginName: b.LoginName, DisplayName: b.DisplayName, Password: b.Password}))
	if err != nil {
		return nil, err
	}
	s.setSession(ctx, sess)
	return gen.AuthApiSetup200JSONResponse(toUser(sess.User)), nil
}

func (s *server) AuthApiSignUp(ctx context.Context, req gen.AuthApiSignUpRequestObject) (gen.AuthApiSignUpResponseObject, error) {
	if !s.opt.Auth().Local {
		return nil, app.ErrDisabled
	}
	sess, err := s.app.Accounts.SignUp(ctx, signUpInput(*req.Body))
	if err != nil {
		return nil, err
	}
	s.setSession(ctx, sess)
	return gen.AuthApiSignUp200JSONResponse(toUser(sess.User)), nil
}

func (s *server) AuthApiLogin(ctx context.Context, req gen.AuthApiLoginRequestObject) (gen.AuthApiLoginResponseObject, error) {
	if !s.opt.Auth().Local {
		return nil, app.ErrDisabled
	}
	sess, err := s.app.Accounts.LoginLocal(ctx, req.Body.LoginName, req.Body.Password)
	if err != nil {
		return nil, err
	}
	s.setSession(ctx, sess)
	return gen.AuthApiLogin200JSONResponse(toUser(sess.User)), nil
}

func (s *server) AuthApiLogout(ctx context.Context, _ gen.AuthApiLogoutRequestObject) (gen.AuthApiLogoutResponseObject, error) {
	if c, err := request(ctx).Cookie(sessionCookie); err == nil {
		if err := s.app.Accounts.Logout(ctx, c.Value); err != nil {
			return nil, err
		}
	}
	s.clearSession(ctx)
	return gen.AuthApiLogout204Response{}, nil
}

func (s *server) MeApiGet(ctx context.Context, _ gen.MeApiGetRequestObject) (gen.MeApiGetResponseObject, error) {
	return gen.MeApiGet200JSONResponse(toUser(currentUser(ctx))), nil
}

func (s *server) MeApiChangePassword(ctx context.Context, req gen.MeApiChangePasswordRequestObject) (gen.MeApiChangePasswordResponseObject, error) {
	if err := s.app.Accounts.ChangePassword(ctx, currentUser(ctx), req.Body.Current, req.Body.Next); err != nil {
		return nil, err
	}
	return gen.MeApiChangePassword204Response{}, nil
}

func (s *server) AdminApiList(ctx context.Context, _ gen.AdminApiListRequestObject) (gen.AdminApiListResponseObject, error) {
	users, err := s.app.Accounts.ListUsers(ctx, currentUser(ctx))
	if err != nil {
		return nil, err
	}
	out := gen.AdminApiList200JSONResponse{}
	for _, u := range users {
		out = append(out, toUser(u))
	}
	return out, nil
}

func (s *server) AdminApiCreate(ctx context.Context, req gen.AdminApiCreateRequestObject) (gen.AdminApiCreateResponseObject, error) {
	b := req.Body
	u, err := s.app.Accounts.CreateUser(ctx, currentUser(ctx), app.SignUpInput{
		LoginName: b.LoginName, DisplayName: str(b.DisplayName), Password: b.Password,
	}, user.Role(b.Role))
	if err != nil {
		return nil, err
	}
	return gen.AdminApiCreate200JSONResponse(toUser(u)), nil
}

func (s *server) AdminApiDelete(ctx context.Context, req gen.AdminApiDeleteRequestObject) (gen.AdminApiDeleteResponseObject, error) {
	if err := s.app.Accounts.DeleteUser(ctx, currentUser(ctx), user.ID(req.Id)); err != nil {
		return nil, err
	}
	return gen.AdminApiDelete204Response{}, nil
}

func (s *server) AdminSourcesApiList(ctx context.Context, _ gen.AdminSourcesApiListRequestObject) (gen.AdminSourcesApiListResponseObject, error) {
	list, err := s.app.Sources.ListFor(ctx, currentUser(ctx))
	if err != nil {
		return nil, err
	}
	out := gen.AdminSourcesApiList200JSONResponse{}
	for _, src := range list {
		out = append(out, toSourceInfo(src))
	}
	return out, nil
}

func (s *server) AdminSourcesApiSync(ctx context.Context, req gen.AdminSourcesApiSyncRequestObject) (gen.AdminSourcesApiSyncResponseObject, error) {
	if err := s.app.Sources.StartSync(ctx, currentUser(ctx), food.DatasetID(req.Id)); err != nil {
		return nil, err
	}
	return gen.AdminSourcesApiSync202Response{}, nil
}

func (s *server) instance() (*app.Instance, error) {
	if s.app.Instance == nil {
		return nil, app.ErrDisabled
	}
	return s.app.Instance, nil
}

func toInstanceSettings(list []app.Setting) []gen.InstanceSetting {
	out := make([]gen.InstanceSetting, 0, len(list))
	for _, st := range list {
		out = append(out, gen.InstanceSetting{
			Key: st.Key, Group: st.Group, Kind: gen.SettingKind(st.Kind), Options: nonNil(st.Options),
			Value: st.Value, DefaultValue: st.Default, EnvValue: st.EnvValue, HasEnv: st.HasEnv,
			Configured: st.Configured, Origin: gen.SettingOrigin(st.Origin),
		})
	}
	return out
}

func (s *server) AdminSettingsApiList(ctx context.Context, _ gen.AdminSettingsApiListRequestObject) (gen.AdminSettingsApiListResponseObject, error) {
	inst, err := s.instance()
	if err != nil {
		return nil, err
	}
	list, err := inst.List(ctx, currentUser(ctx))
	if err != nil {
		return nil, err
	}
	return gen.AdminSettingsApiList200JSONResponse(toInstanceSettings(list)), nil
}

func (s *server) AdminSettingsApiUpdate(ctx context.Context, req gen.AdminSettingsApiUpdateRequestObject) (gen.AdminSettingsApiUpdateResponseObject, error) {
	inst, err := s.instance()
	if err != nil {
		return nil, err
	}
	list, err := inst.Update(ctx, currentUser(ctx), req.Body.Values)
	if err != nil {
		return nil, err
	}
	return gen.AdminSettingsApiUpdate200JSONResponse(toInstanceSettings(list)), nil
}

func (s *server) AdminSettingsApiResetAll(ctx context.Context, _ gen.AdminSettingsApiResetAllRequestObject) (gen.AdminSettingsApiResetAllResponseObject, error) {
	inst, err := s.instance()
	if err != nil {
		return nil, err
	}
	list, err := inst.Reset(ctx, currentUser(ctx), nil)
	if err != nil {
		return nil, err
	}
	return gen.AdminSettingsApiResetAll200JSONResponse(toInstanceSettings(list)), nil
}

func (s *server) AdminSettingsApiReset(ctx context.Context, req gen.AdminSettingsApiResetRequestObject) (gen.AdminSettingsApiResetResponseObject, error) {
	inst, err := s.instance()
	if err != nil {
		return nil, err
	}
	list, err := inst.Reset(ctx, currentUser(ctx), []string{req.Key})
	if err != nil {
		return nil, err
	}
	return gen.AdminSettingsApiReset200JSONResponse(toInstanceSettings(list)), nil
}

// 食品

func (s *server) FoodsApiSearchExternal(ctx context.Context, req gen.FoodsApiSearchExternalRequestObject) (gen.FoodsApiSearchExternalResponseObject, error) {
	res, err := s.app.Sources.Search(ctx, req.Params.Q)
	if err != nil {
		return nil, err
	}
	out := gen.FoodsApiSearchExternal200JSONResponse{Foods: []gen.ExternalFood{}, Failed: []string{}, Attributions: []string{}}
	attributed := map[string]bool{}
	for _, h := range res.Hits {
		out.Foods = append(out.Foods, toExternalFood(h))
		if a := h.Dataset.Attribution; a != "" && !attributed[a] {
			attributed[a] = true
			out.Attributions = append(out.Attributions, a)
		}
	}
	for _, id := range res.Failed {
		out.Failed = append(out.Failed, string(id))
	}
	return out, nil
}

func (s *server) FoodsApiSaveExternal(ctx context.Context, req gen.FoodsApiSaveExternalRequestObject) (gen.FoodsApiSaveExternalResponseObject, error) {
	u := currentUser(ctx)
	id, err := s.app.Sources.Save(ctx, food.DatasetID(req.Body.DatasetId), req.Body.Reference)
	if err != nil {
		return nil, err
	}
	v, err := s.app.Foods.Get(ctx, u.ID, id)
	if err != nil {
		return nil, err
	}
	return gen.FoodsApiSaveExternal200JSONResponse(toFood(v, u.ID)), nil
}

func (s *server) FoodsApiSearch(ctx context.Context, req gen.FoodsApiSearchRequestObject) (gen.FoodsApiSearchResponseObject, error) {
	u := currentUser(ctx)
	var kinds []food.Kind
	if req.Params.Kind != nil {
		for _, k := range *req.Params.Kind {
			kinds = append(kinds, food.Kind(k))
		}
	}
	limit, offset := 30, 0
	if req.Params.Limit != nil {
		limit = int(*req.Params.Limit)
	}
	if req.Params.Offset != nil {
		offset = int(*req.Params.Offset)
	}
	views, err := s.app.Foods.Search(ctx, u.ID, req.Params.Q, kinds, limit, offset)
	if err != nil {
		return nil, err
	}
	return gen.FoodsApiSearch200JSONResponse(toFoods(views, u.ID)), nil
}

func (s *server) FoodsApiCreate(ctx context.Context, req gen.FoodsApiCreateRequestObject) (gen.FoodsApiCreateResponseObject, error) {
	u := currentUser(ctx)
	in, err := fromFoodInput(*req.Body)
	if err != nil {
		return nil, app.Invalid(err)
	}
	v, err := s.app.Foods.Create(ctx, u.ID, in)
	if err != nil {
		return nil, err
	}
	return gen.FoodsApiCreate200JSONResponse(toFood(v, u.ID)), nil
}

func (s *server) FoodsApiListMine(ctx context.Context, _ gen.FoodsApiListMineRequestObject) (gen.FoodsApiListMineResponseObject, error) {
	u := currentUser(ctx)
	views, err := s.app.Foods.ListOwned(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return gen.FoodsApiListMine200JSONResponse(toFoods(views, u.ID)), nil
}

func (s *server) FoodsApiGetByBarcode(ctx context.Context, req gen.FoodsApiGetByBarcodeRequestObject) (gen.FoodsApiGetByBarcodeResponseObject, error) {
	u := currentUser(ctx)
	v, err := s.app.Foods.ByBarcode(ctx, u.ID, req.Params.Code)
	if err != nil {
		return nil, err
	}
	return gen.FoodsApiGetByBarcode200JSONResponse(toFood(v, u.ID)), nil
}

func (s *server) FoodsApiGet(ctx context.Context, req gen.FoodsApiGetRequestObject) (gen.FoodsApiGetResponseObject, error) {
	u := currentUser(ctx)
	v, err := s.app.Foods.Get(ctx, u.ID, food.ID(req.Id))
	if err != nil {
		return nil, err
	}
	return gen.FoodsApiGet200JSONResponse(toFood(v, u.ID)), nil
}

func (s *server) FoodsApiUpdate(ctx context.Context, req gen.FoodsApiUpdateRequestObject) (gen.FoodsApiUpdateResponseObject, error) {
	u := currentUser(ctx)
	in, err := fromFoodInput(*req.Body)
	if err != nil {
		return nil, app.Invalid(err)
	}
	v, err := s.app.Foods.Update(ctx, u.ID, food.ID(req.Id), in)
	if err != nil {
		return nil, err
	}
	return gen.FoodsApiUpdate200JSONResponse(toFood(v, u.ID)), nil
}

func (s *server) FoodsApiDelete(ctx context.Context, req gen.FoodsApiDeleteRequestObject) (gen.FoodsApiDeleteResponseObject, error) {
	if err := s.app.Foods.Delete(ctx, currentUser(ctx).ID, food.ID(req.Id)); err != nil {
		return nil, err
	}
	return gen.FoodsApiDelete204Response{}, nil
}

func (s *server) FoodsApiListRevisions(ctx context.Context, req gen.FoodsApiListRevisionsRequestObject) (gen.FoodsApiListRevisionsResponseObject, error) {
	revs, err := s.app.Foods.Revisions(ctx, currentUser(ctx).ID, food.ID(req.Id))
	if err != nil {
		return nil, err
	}
	out := gen.FoodsApiListRevisions200JSONResponse{}
	for _, r := range revs {
		out = append(out, toRevision(r))
	}
	return out, nil
}

func (s *server) FoodsApiRecordSelection(ctx context.Context, req gen.FoodsApiRecordSelectionRequestObject) (gen.FoodsApiRecordSelectionResponseObject, error) {
	if err := s.app.Foods.RecordSelection(ctx, currentUser(ctx).ID, req.Body.Query, food.ID(req.Id)); err != nil {
		return nil, err
	}
	return gen.FoodsApiRecordSelection204Response{}, nil
}

func (s *server) FoodsApiAddFavorite(ctx context.Context, req gen.FoodsApiAddFavoriteRequestObject) (gen.FoodsApiAddFavoriteResponseObject, error) {
	if err := s.app.Foods.SetFavorite(ctx, currentUser(ctx).ID, food.ID(req.Id), true); err != nil {
		return nil, err
	}
	return gen.FoodsApiAddFavorite204Response{}, nil
}

func (s *server) FoodsApiRemoveFavorite(ctx context.Context, req gen.FoodsApiRemoveFavoriteRequestObject) (gen.FoodsApiRemoveFavoriteResponseObject, error) {
	if err := s.app.Foods.SetFavorite(ctx, currentUser(ctx).ID, food.ID(req.Id), false); err != nil {
		return nil, err
	}
	return gen.FoodsApiRemoveFavorite204Response{}, nil
}

func (s *server) FavoritesApiList(ctx context.Context, _ gen.FavoritesApiListRequestObject) (gen.FavoritesApiListResponseObject, error) {
	u := currentUser(ctx)
	views, err := s.app.Foods.Favorites(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return gen.FavoritesApiList200JSONResponse(toFoods(views, u.ID)), nil
}

func (s *server) SearchHistoryApiList(ctx context.Context, _ gen.SearchHistoryApiListRequestObject) (gen.SearchHistoryApiListResponseObject, error) {
	list, err := s.app.Foods.SearchHistory(ctx, currentUser(ctx).ID, 20)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []string{}
	}
	return gen.SearchHistoryApiList200JSONResponse(list), nil
}

func (s *server) SearchHistoryApiClear(ctx context.Context, _ gen.SearchHistoryApiClearRequestObject) (gen.SearchHistoryApiClearResponseObject, error) {
	if err := s.app.Foods.ClearSearchHistory(ctx, currentUser(ctx).ID); err != nil {
		return nil, err
	}
	return gen.SearchHistoryApiClear204Response{}, nil
}

func (s *server) RecipesApiCreate(ctx context.Context, req gen.RecipesApiCreateRequestObject) (gen.RecipesApiCreateResponseObject, error) {
	u := currentUser(ctx)
	v, err := s.app.Recipes.Create(ctx, u.ID, fromRecipeInput(*req.Body))
	if err != nil {
		return nil, err
	}
	return gen.RecipesApiCreate200JSONResponse(toRecipe(v, u.ID)), nil
}

func (s *server) RecipesApiGet(ctx context.Context, req gen.RecipesApiGetRequestObject) (gen.RecipesApiGetResponseObject, error) {
	u := currentUser(ctx)
	v, err := s.app.Recipes.Get(ctx, u.ID, food.ID(req.Id))
	if err != nil {
		return nil, err
	}
	return gen.RecipesApiGet200JSONResponse(toRecipe(v, u.ID)), nil
}

func (s *server) RecipesApiUpdate(ctx context.Context, req gen.RecipesApiUpdateRequestObject) (gen.RecipesApiUpdateResponseObject, error) {
	u := currentUser(ctx)
	v, err := s.app.Recipes.Update(ctx, u.ID, food.ID(req.Id), fromRecipeInput(*req.Body))
	if err != nil {
		return nil, err
	}
	return gen.RecipesApiUpdate200JSONResponse(toRecipe(v, u.ID)), nil
}

// 食事記録

func (s *server) DaysApiGet(ctx context.Context, req gen.DaysApiGetRequestObject) (gen.DaysApiGetResponseObject, error) {
	u := currentUser(ctx)
	date := fromDate(req.Date)
	entries, err := s.app.Meals.Day(ctx, u.ID, date)
	if err != nil {
		return nil, err
	}
	report, err := s.app.Reports.Daily(ctx, u.ID, date)
	if err != nil {
		return nil, err
	}
	out := gen.DaysApiGet200JSONResponse{
		Date:    req.Date,
		Entries: toEntries(entries),
		Report:  toDailyReport(report),
	}
	if w, ok, err := s.app.Weights.LatestOnOrBefore(ctx, u.ID, date); err != nil {
		return nil, err
	} else if ok {
		gw := toWeight(w)
		out.LatestWeight = &gw
	}
	return out, nil
}

func (s *server) MealsApiAdd(ctx context.Context, req gen.MealsApiAddRequestObject) (gen.MealsApiAddResponseObject, error) {
	b := req.Body
	e, err := s.app.Meals.Add(ctx, currentUser(ctx).ID, app.AddEntryInput{
		Date:      fromDate(b.Date),
		Slot:      meal.Slot(b.Slot),
		FoodID:    food.ID(b.FoodId),
		Quantity:  fromQuantity(b.Quantity),
		ClientRef: str(b.ClientRef),
	})
	if err != nil {
		return nil, err
	}
	return gen.MealsApiAdd200JSONResponse(toEntry(e)), nil
}

func (s *server) MealsApiQuickPicks(ctx context.Context, _ gen.MealsApiQuickPicksRequestObject) (gen.MealsApiQuickPicksResponseObject, error) {
	u := currentUser(ctx)
	q, err := s.app.Meals.QuickPicks(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	return gen.MealsApiQuickPicks200JSONResponse(toQuickPicks(q, u.ID)), nil
}

func (s *server) MealsApiCopy(ctx context.Context, req gen.MealsApiCopyRequestObject) (gen.MealsApiCopyResponseObject, error) {
	b := req.Body
	in := app.CopyInput{FromDate: fromDate(b.FromDate), ToDate: fromDate(b.ToDate)}
	if b.FromSlot != nil {
		slot := meal.Slot(*b.FromSlot)
		in.FromSlot = &slot
	}
	if b.ToSlot != nil {
		slot := meal.Slot(*b.ToSlot)
		in.ToSlot = &slot
	}
	entries, err := s.app.Meals.Copy(ctx, currentUser(ctx).ID, in)
	if err != nil {
		return nil, err
	}
	return gen.MealsApiCopy200JSONResponse(toEntries(entries)), nil
}

func (s *server) MealsApiUpdate(ctx context.Context, req gen.MealsApiUpdateRequestObject) (gen.MealsApiUpdateResponseObject, error) {
	b := req.Body
	var in app.UpdateEntryInput
	if b.Date != nil {
		d := fromDate(*b.Date)
		in.Date = &d
	}
	if b.Slot != nil {
		slot := meal.Slot(*b.Slot)
		in.Slot = &slot
	}
	if b.Quantity != nil {
		q := fromQuantity(*b.Quantity)
		in.Quantity = &q
	}
	e, err := s.app.Meals.Update(ctx, currentUser(ctx).ID, meal.ID(req.Id), in)
	if err != nil {
		return nil, err
	}
	return gen.MealsApiUpdate200JSONResponse(toEntry(e)), nil
}

func (s *server) MealsApiDelete(ctx context.Context, req gen.MealsApiDeleteRequestObject) (gen.MealsApiDeleteResponseObject, error) {
	if err := s.app.Meals.Delete(ctx, currentUser(ctx).ID, meal.ID(req.Id)); err != nil {
		return nil, err
	}
	return gen.MealsApiDelete204Response{}, nil
}

func (s *server) MealsApiRecalculate(ctx context.Context, req gen.MealsApiRecalculateRequestObject) (gen.MealsApiRecalculateResponseObject, error) {
	e, err := s.app.Meals.Recalculate(ctx, currentUser(ctx).ID, meal.ID(req.Id))
	if err != nil {
		return nil, err
	}
	return gen.MealsApiRecalculate200JSONResponse(toEntry(e)), nil
}

// 集計・設定

func (s *server) ReportsApiDaily(ctx context.Context, req gen.ReportsApiDailyRequestObject) (gen.ReportsApiDailyResponseObject, error) {
	r, err := s.app.Reports.Daily(ctx, currentUser(ctx).ID, fromDate(req.Date))
	if err != nil {
		return nil, err
	}
	return gen.ReportsApiDaily200JSONResponse(toDailyReport(r)), nil
}

func (s *server) ReportsApiRange(ctx context.Context, req gen.ReportsApiRangeRequestObject) (gen.ReportsApiRangeResponseObject, error) {
	r, err := s.app.Reports.Range(ctx, currentUser(ctx).ID, fromDate(req.Params.From), fromDate(req.Params.To))
	if err != nil {
		return nil, err
	}
	return gen.ReportsApiRange200JSONResponse(toRangeReport(r)), nil
}

func (s *server) SettingsApiGet(ctx context.Context, _ gen.SettingsApiGetRequestObject) (gen.SettingsApiGetResponseObject, error) {
	st, err := s.app.Settings.Current(ctx, currentUser(ctx).ID)
	if err != nil {
		return nil, err
	}
	return gen.SettingsApiGet200JSONResponse(toSettings(st)), nil
}

func (s *server) SettingsApiSave(ctx context.Context, req gen.SettingsApiSaveRequestObject) (gen.SettingsApiSaveResponseObject, error) {
	b := req.Body
	in := app.SettingsInput{Profile: fromProfileInput(b.Profile)}
	if b.EffectiveFrom != nil {
		in.EffectiveFrom = fromDate(*b.EffectiveFrom)
	}
	if b.Overrides != nil {
		in.Overrides = fromOverrides(*b.Overrides)
	}
	st, err := s.app.Settings.Save(ctx, currentUser(ctx).ID, in)
	if err != nil {
		return nil, err
	}
	return gen.SettingsApiSave200JSONResponse(toSettings(st)), nil
}

func (s *server) SettingsApiHistory(ctx context.Context, _ gen.SettingsApiHistoryRequestObject) (gen.SettingsApiHistoryResponseObject, error) {
	list, err := s.app.Settings.History(ctx, currentUser(ctx).ID)
	if err != nil {
		return nil, err
	}
	out := gen.SettingsApiHistory200JSONResponse{}
	for _, st := range list {
		out = append(out, toSettings(st))
	}
	return out, nil
}

func (s *server) TargetsApiGet(ctx context.Context, req gen.TargetsApiGetRequestObject) (gen.TargetsApiGetResponseObject, error) {
	ts, err := s.app.Settings.TargetsAt(ctx, currentUser(ctx).ID, fromDate(req.Date))
	if err != nil {
		return nil, err
	}
	return gen.TargetsApiGet200JSONResponse(toTargets(ts)), nil
}

func (s *server) WeightsApiList(ctx context.Context, req gen.WeightsApiListRequestObject) (gen.WeightsApiListResponseObject, error) {
	ws, err := s.app.Weights.Range(ctx, currentUser(ctx).ID, fromDate(req.Params.From), fromDate(req.Params.To))
	if err != nil {
		return nil, err
	}
	return gen.WeightsApiList200JSONResponse(toWeights(ws)), nil
}

func (s *server) WeightsApiSave(ctx context.Context, req gen.WeightsApiSaveRequestObject) (gen.WeightsApiSaveResponseObject, error) {
	w, err := s.app.Weights.Save(ctx, currentUser(ctx).ID, body.Weight{
		Date: fromDate(req.Date), Kg: dec(req.Body.Kg), BodyFatPct: decp(req.Body.BodyFatPct),
	})
	if err != nil {
		return nil, err
	}
	return gen.WeightsApiSave200JSONResponse(toWeight(w)), nil
}

func (s *server) WeightsApiDelete(ctx context.Context, req gen.WeightsApiDeleteRequestObject) (gen.WeightsApiDeleteResponseObject, error) {
	if err := s.app.Weights.Delete(ctx, currentUser(ctx).ID, fromDate(req.Date)); err != nil {
		return nil, err
	}
	return gen.WeightsApiDelete204Response{}, nil
}

// 参照データ

func (s *server) ReferenceApiNutrients(_ context.Context, _ gen.ReferenceApiNutrientsRequestObject) (gen.ReferenceApiNutrientsResponseObject, error) {
	out := gen.ReferenceApiNutrients200JSONResponse{}
	for _, d := range nutrient.Catalog() {
		out = append(out, gen.NutrientDefinition{Code: string(d.Code), Unit: gen.NutrientUnit(d.Unit), Group: gen.NutrientGroup(d.Group)})
	}
	return out, nil
}

func (s *server) ReferenceApiDatasets(ctx context.Context, _ gen.ReferenceApiDatasetsRequestObject) (gen.ReferenceApiDatasetsResponseObject, error) {
	list, err := s.app.Foods.Datasets(ctx)
	if err != nil {
		return nil, err
	}
	out := gen.ReferenceApiDatasets200JSONResponse{}
	for _, d := range list {
		out = append(out, toDataset(d))
	}
	return out, nil
}

func (s *server) ReferenceApiDietaryReference(_ context.Context, _ gen.ReferenceApiDietaryReferenceRequestObject) (gen.ReferenceApiDietaryReferenceResponseObject, error) {
	ref := s.app.Settings.Reference()
	return gen.ReferenceApiDietaryReference200JSONResponse{Edition: ref.Edition, Title: ref.Title, Url: ref.URL}, nil
}

// エクスポート

func attachment(name string) string {
	return fmt.Sprintf(`attachment; filename="%s"`, name)
}

func (s *server) ExportApiJson(ctx context.Context, _ gen.ExportApiJsonRequestObject) (gen.ExportApiJsonResponseObject, error) {
	b, err := s.app.Exports.Export(ctx, currentUser(ctx))
	if err != nil {
		return nil, err
	}
	return gen.ExportApiJson200JSONResponse{
		Body:    ToBundle(b),
		Headers: gen.ExportApiJson200ResponseHeaders{ContentDisposition: attachment("mecalo-" + b.ExportedAt.Format("20060102") + ".json")},
	}, nil
}

func (s *server) ExportApiMealsCsv(ctx context.Context, _ gen.ExportApiMealsCsvRequestObject) (gen.ExportApiMealsCsvResponseObject, error) {
	b, err := s.app.Exports.Export(ctx, currentUser(ctx))
	if err != nil {
		return nil, err
	}
	data, err := MealsCSV(b.Meals)
	if err != nil {
		return nil, err
	}
	return gen.ExportApiMealsCsv200TextcsvResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
		Headers:       gen.ExportApiMealsCsv200ResponseHeaders{ContentDisposition: attachment("mecalo-meals.csv")},
	}, nil
}

func (s *server) ExportApiWeightsCsv(ctx context.Context, _ gen.ExportApiWeightsCsvRequestObject) (gen.ExportApiWeightsCsvResponseObject, error) {
	b, err := s.app.Exports.Export(ctx, currentUser(ctx))
	if err != nil {
		return nil, err
	}
	data, err := WeightsCSV(b.Weights)
	if err != nil {
		return nil, err
	}
	return gen.ExportApiWeightsCsv200TextcsvResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
		Headers:       gen.ExportApiWeightsCsv200ResponseHeaders{ContentDisposition: attachment("mecalo-weights.csv")},
	}, nil
}

func (s *server) ImportApiJson(ctx context.Context, req gen.ImportApiJsonRequestObject) (gen.ImportApiJsonResponseObject, error) {
	u := currentUser(ctx)
	b, err := FromBundle(u.ID, *req.Body)
	if err != nil {
		return nil, app.Invalid(err)
	}
	res, err := s.app.Exports.Import(ctx, u.ID, b)
	if err != nil {
		return nil, err
	}
	return gen.ImportApiJson200JSONResponse{
		Foods: int32(res.Foods), Meals: int32(res.Meals), Weights: int32(res.Weights),
		Settings: int32(res.Settings), Skipped: int32(res.Skipped),
	}, nil
}

func (s *server) HealthApiGet(_ context.Context, _ gen.HealthApiGetRequestObject) (gen.HealthApiGetResponseObject, error) {
	return gen.HealthApiGet200JSONResponse{Status: gen.Ok}, nil
}

// 食事記録を1行1記録のCSVへ, 栄養素は未計測を空欄
func MealsCSV(entries []meal.Entry) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	header := []string{"date", "slot", "food_name", "brand", "quantity", "unit", "grams", "dataset", "source_ref"}
	catalog := nutrient.Catalog()
	for _, d := range catalog {
		header = append(header, fmt.Sprintf("%s_%s", d.Code, d.Unit))
	}
	if err := w.Write(header); err != nil {
		return nil, err
	}
	for _, e := range entries {
		row := []string{
			e.Date.String(), string(e.Slot), e.FoodName, e.Brand, e.Quantity.Amount.String(), e.Quantity.Unit,
			e.Grams.String(), string(e.Source.DatasetID), e.Source.Reference,
		}
		for _, d := range catalog {
			v := e.Nutrients.Get(d.Code)
			if v.Known() {
				row = append(row, v.Amount.Round(4).String())
			} else {
				row = append(row, "")
			}
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}

func WeightsCSV(weights []body.Weight) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write([]string{"date", "kg", "body_fat_pct"}); err != nil {
		return nil, err
	}
	for _, wt := range weights {
		fat := ""
		if wt.BodyFatPct != nil {
			fat = wt.BodyFatPct.String()
		}
		if err := w.Write([]string{wt.Date.String(), wt.Kg.String(), fat}); err != nil {
			return nil, err
		}
	}
	w.Flush()
	return buf.Bytes(), w.Error()
}
