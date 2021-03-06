package client

import (
	// Stdlib
	"log"
	"strconv"
	"strings"
	"time"

	// Vendor
	"github.com/pkg/errors"

	// RPC
	"github.com/asuleymanov/rpc/encoding/wif"
	"github.com/asuleymanov/rpc/transactions"
	"github.com/asuleymanov/rpc/translit"
	"github.com/asuleymanov/rpc/types"
)

func (api *Client) SteemPerMvest() (float64, error) {
	dgp, errdgp := api.Rpc.Database.GetDynamicGlobalProperties()
	if errdgp != nil {
		return 0, errdgp
	}

	tvfs, errtvfs := strconv.ParseFloat(strings.Split(dgp.TotalVersingFundSteem, " ")[0], 64)
	if errtvfs != nil {
		return 0, errtvfs
	}

	tvs, errtvs := strconv.ParseFloat(strings.Split(dgp.TotalVestingShares, " ")[0], 64)
	if errtvs != nil {
		return 0, errtvs
	}

	spmtmp := (tvfs / tvs) * 1000000
	str := strconv.FormatFloat(spmtmp, 'f', 3, 64)

	spm, errspm := strconv.ParseFloat(str, 64)
	if errspm != nil {
		return 0, errspm
	}

	return spm, nil
}

func (api *Client) Vote(user_name, author_name, permlink string, weight int) error {
	if weight > 10000 {
		weight = 10000
	}
	if api.Verify_Voter_Weight(author_name, permlink, user_name, weight) {
		return errors.New("The voter is on the list")
	}
	tx := &types.VoteOperation{
		Voter:    user_name,
		Author:   author_name,
		Permlink: permlink,
		Weight:   types.Int16(weight),
	}
	resp, err := api.Send_Trx(user_name, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Vote: ")
	} else {
		log.Println("[Vote] Block -> ", resp.BlockNum, " User -> ", user_name)
		return nil
	}
}

func (api *Client) Comment(user_name, author_name, ppermlink, body string, v *PC_Vote, o *PC_Options) error {
	var trx []types.Operation

	times, _ := strconv.Unquote(time.Now().Add(30 * time.Second).UTC().Format(fdt))
	permlink := "re-" + author_name + "-" + ppermlink + "-" + times
	permlink = strings.Replace(permlink, ".", "-", -1)

	tx := &types.CommentOperation{
		ParentAuthor:   author_name,
		ParentPermlink: ppermlink,
		Author:         user_name,
		Permlink:       permlink,
		Title:          "",
		Body:           body,
		JsonMetadata:   "{\"app\":\"steem-go\"}",
	}
	trx = append(trx, tx)

	if o != nil {
		symbol := "SBD"
		MAP := "1000000.000 " + symbol
		PSD := o.Percent
		if o.Percent == 0 {
			MAP = "0.000 " + symbol
			PSD = 10000
		} else if o.Percent == 50 {
			PSD = 10000
		} else {
			PSD = 0
		}

		var ext []interface{}
		if len(o.BenefList) > 0 {
			var ben_list []types.Beneficiarie
			var benef types.CommentPayoutBeneficiaries
			for _, val := range o.BenefList {
				ben_list = append(ben_list, types.Beneficiarie{val.Account, val.Weight})
			}
			benef.Beneficiaries = ben_list
			ext = append(ext, 0)
			ext = append(ext, benef)
		}
		Extens := []interface{}{}
		if len(ext) > 0 {
			Extens = []interface{}{ext}
		}

		txo := &types.CommentOptionsOperation{
			Author:               user_name,
			Permlink:             permlink,
			MaxAcceptedPayout:    MAP,
			PercentSteemDollars:  PSD,
			AllowVotes:           true,
			AllowCurationRewards: true,
			Extensions:           Extens,
		}
		trx = append(trx, txo)
	}

	if v != nil && v.Weight != 0 {
		txv := &types.VoteOperation{
			Voter:    user_name,
			Author:   user_name,
			Permlink: permlink,
			Weight:   types.Int16(v.Weight),
		}
		trx = append(trx, txv)
	}

	resp, err := api.Send_Arr_Trx(user_name, trx)
	if err != nil {
		return errors.Wrapf(err, "Error Comment : ")
	} else {
		log.Println("[Comment] Block -> ", resp.BlockNum, " User -> ", user_name)
		return nil
	}
}

func (api *Client) DeleteComment(author_name, permlink string) error {
	if api.Verify_Votes(author_name, permlink) {
		return errors.New("You can not delete already there are voted")
	}
	if api.Verify_Comments(author_name, permlink) {
		return errors.New("You can not delete already have comments")
	}
	tx := &types.DeleteCommentOperation{
		Author:   author_name,
		Permlink: permlink,
	}
	resp, err := api.Send_Trx(author_name, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Delete Comment: ")
	} else {
		log.Println("[Delete Comment] Block -> ", resp.BlockNum, " User -> ", author_name)
		return nil
	}
}

func (api *Client) Post(author_name, title, body, permlink, ptag, post_image string, tags []string, v *PC_Vote, o *PC_Options) error {
	if permlink == "" {
		permlink = translit.EncodeTitle(title)
	} else {
		permlink = translit.EncodeTitle(permlink)
	}
	tag := translit.EncodeTags(tags)
	if ptag == "" {
		ptag = translit.EncodeTag(tags[0])
	} else {
		ptag = translit.EncodeTag(ptag)
	}

	json_meta := "{\"tags\":["
	for k, v := range tag {
		if k != len(tags)-1 {
			json_meta = json_meta + "\"" + v + "\","
		} else {
			json_meta = json_meta + "\"" + v + "\"]"
		}
	}
	if post_image != "" {
		json_meta = json_meta + ",\"image\":[\"" + post_image + "\"]"
	}
	json_meta = json_meta + ",\"app\":\"steem-go\"}"

	var trx []types.Operation
	txp := &types.CommentOperation{
		ParentAuthor:   "",
		ParentPermlink: ptag,
		Author:         author_name,
		Permlink:       permlink,
		Title:          title,
		Body:           body,
		JsonMetadata:   json_meta,
	}
	trx = append(trx, txp)

	if o != nil {
		symbol := "SBD"
		MAP := "1000000.000 " + symbol
		PSD := o.Percent
		if o.Percent == 0 {
			MAP = "0.000 " + symbol
			PSD = 10000
		} else if o.Percent == 50 {
			PSD = 10000
		} else {
			PSD = 0
		}

		var ext []interface{}
		if len(o.BenefList) > 0 {
			var ben_list []types.Beneficiarie
			var benef types.CommentPayoutBeneficiaries
			for _, val := range o.BenefList {
				ben_list = append(ben_list, types.Beneficiarie{val.Account, val.Weight})
			}
			benef.Beneficiaries = ben_list
			ext = append(ext, 0)
			ext = append(ext, benef)
		}
		Extens := []interface{}{}
		if len(ext) > 0 {
			Extens = []interface{}{ext}
		}

		txo := &types.CommentOptionsOperation{
			Author:               author_name,
			Permlink:             permlink,
			MaxAcceptedPayout:    MAP,
			PercentSteemDollars:  PSD,
			AllowVotes:           true,
			AllowCurationRewards: true,
			Extensions:           Extens,
		}
		trx = append(trx, txo)
	}

	if v != nil && v.Weight != 0 {
		txv := &types.VoteOperation{
			Voter:    author_name,
			Author:   author_name,
			Permlink: permlink,
			Weight:   types.Int16(v.Weight),
		}
		trx = append(trx, txv)
	}

	resp, err := api.Send_Arr_Trx(author_name, trx)
	if err != nil {
		return errors.Wrapf(err, "Error Post : ")
	} else {
		log.Println("[Post] Block -> ", resp.BlockNum, " User -> ", author_name)
		return nil
	}
}

func (api *Client) Follow(follower, following string) error {
	json_string := "[\"follow\",{\"follower\":\"" + follower + "\",\"following\":\"" + following + "\",\"what\":[\"blog\"]}]"

	tx := &types.CustomJSONOperation{
		RequiredAuths:        []string{},
		RequiredPostingAuths: []string{follower},
		ID:                   "follow",
		JSON:                 json_string,
	}
	resp, err := api.Send_Trx(follower, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Follow: ")
	} else {
		log.Println("[Follow] Block -> ", resp.BlockNum, " Follower user -> ", follower, " Following user -> ", following)
		return nil
	}
}

func (api *Client) Unfollow(follower, following string) error {
	json_string := "[\"follow\",{\"follower\":\"" + follower + "\",\"following\":\"" + following + "\",\"what\":[]}]"

	tx := &types.CustomJSONOperation{
		RequiredAuths:        []string{},
		RequiredPostingAuths: []string{follower},
		ID:                   "follow",
		JSON:                 json_string,
	}
	resp, err := api.Send_Trx(follower, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Unfollow: ")
	} else {
		log.Println("[Unfollow] Block -> ", resp.BlockNum, " Unfollower user -> ", follower, " Unfollowing user -> ", following)
		return nil
	}
}

func (api *Client) Ignore(follower, following string) error {
	json_string := "[\"follow\",{\"follower\":\"" + follower + "\",\"following\":\"" + following + "\",\"what\":[\"ignore\"]}]"

	tx := &types.CustomJSONOperation{
		RequiredAuths:        []string{},
		RequiredPostingAuths: []string{follower},
		ID:                   "follow",
		JSON:                 json_string,
	}
	resp, err := api.Send_Trx(follower, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Ignore: ")
	} else {
		log.Println("[Ignore] Block -> ", resp.BlockNum, " Ignore user -> ", follower, " Ignoring user -> ", following)
		return nil
	}
}

func (api *Client) Notice(follower, following string) error {
	json_string := "[\"follow\",{\"follower\":\"" + follower + "\",\"following\":\"" + following + "\",\"what\":[]}]"

	tx := &types.CustomJSONOperation{
		RequiredAuths:        []string{},
		RequiredPostingAuths: []string{follower},
		ID:                   "follow",
		JSON:                 json_string,
	}
	resp, err := api.Send_Trx(follower, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Notice: ")
	} else {
		log.Println("[Notice] Block -> ", resp.BlockNum, " Notice user -> ", follower, " Noticing user -> ", following)
		return nil
	}
}

func (api *Client) Reblog(user_name, author_name, permlink string) error {
	if api.Verify_Reblogs(author_name, permlink, user_name) {
		return errors.New("The user already did repost")
	}
	json_string := "[\"reblog\",{\"account\":\"" + user_name + "\",\"author\":\"" + author_name + "\",\"permlink\":\"" + permlink + "\"}]"

	tx := &types.CustomJSONOperation{
		RequiredAuths:        []string{},
		RequiredPostingAuths: []string{user_name},
		ID:                   "follow",
		JSON:                 json_string,
	}
	resp, err := api.Send_Trx(user_name, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Reblog: ")
	} else {
		log.Println("[Reblog] Block -> ", resp.BlockNum, " Reblog user -> ", user_name, " Rebloging -> ", author_name, "/", permlink)
		return nil
	}
}

func (api *Client) AccountWitnessVote(user_name, witness_name string, approv bool) error {
	tx := &types.AccountWitnessVoteOperation{
		Account: user_name,
		Witness: witness_name,
		Approve: approv,
	}
	resp, err := api.Send_Trx(user_name, tx)
	if err != nil {
		return errors.Wrapf(err, "Error AccountWitnessVote: ")
	} else {
		log.Println("[AccountWitnessVote] Block -> ", resp.BlockNum, " User -> ", user_name)
		return nil
	}
}

func (api *Client) AccountWitnessProxy(user_name, proxy string) error {
	tx := &types.AccountWitnessProxyOperation{
		Account: user_name,
		Proxy:   proxy,
	}
	resp, err := api.Send_Trx(user_name, tx)
	if err != nil {
		return errors.Wrapf(err, "Error AccountWitnessProxy: ")
	} else {
		log.Println("[AccountWitnessProxy] Block -> ", resp.BlockNum, " User -> ", user_name)
		return nil
	}
}

func (api *Client) Transfer(from_name, to_name, memo, ammount string) error {
	tx := &types.TransferOperation{
		From:   from_name,
		To:     to_name,
		Amount: ammount,
		Memo:   memo,
	}
	resp, err := api.Send_Trx(from_name, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Transfer: ")
	} else {
		log.Println("[Transfer] Block -> ", resp.BlockNum, " From user -> ", from_name, " To user -> ", to_name)
		return nil
	}
}

func (api *Client) Multi_Transfer(username string, arrtrans []ArrTransfer) error {
	var trx []types.Operation

	for _, val := range arrtrans {
		txt := &types.TransferOperation{
			From:   username,
			To:     val.To,
			Amount: val.Ammount,
			Memo:   val.Memo,
		}
		trx = append(trx, txt)
	}

	resp, err := api.Send_Arr_Trx(username, trx)
	if err != nil {
		return errors.Wrapf(err, "Error Multi_Transfer: ")
	} else {
		log.Println("[Multi_Transfer] Block -> ", resp.BlockNum, " From user -> ", username)
		return nil
	}
}

func (api *Client) Login(user_name, key string) bool {
	json_string := "[\"login\",{\"account\":\"" + user_name + "\",\"app\":\"steem-go\"}]"

	strx := &types.CustomJSONOperation{
		RequiredAuths:        []string{},
		RequiredPostingAuths: []string{user_name},
		ID:                   "login",
		JSON:                 json_string,
	}

	props, err := api.Rpc.Database.GetDynamicGlobalProperties()
	if err != nil {
		log.Println("GDP")
		return false
	}

	// Создание транзакции
	refBlockPrefix, err := transactions.RefBlockPrefix(props.HeadBlockID)
	if err != nil {
		log.Println("Create")
		return false
	}
	tx := transactions.NewSignedTransaction(&types.Transaction{
		RefBlockNum:    transactions.RefBlockNum(props.HeadBlockNumber),
		RefBlockPrefix: refBlockPrefix,
	})

	// Добавление операций в транзакцию
	tx.PushOperation(strx)

	// Получаем необходимый для подписи ключ
	var keys [][]byte
	privKey, _ := wif.Decode(string([]byte(key)))
	keys = append(keys, privKey)

	// Подписываем транзакцию
	if err := tx.Sign(keys, api.Chain); err != nil {
		log.Println("Sign")
		return false
	}

	// Отправка транзакции
	resp, err := api.Rpc.NetworkBroadcast.BroadcastTransactionSynchronous(tx.Transaction)

	if err != nil {
		log.Println(err)
		return false
	} else {
		log.Println("[Login] Block -> ", resp.BlockNum, " User -> ", user_name)
		return true
	}
}

func (api *Client) LimitOrderCancel(owner string, orderid uint32) error {

	tx := &types.LimitOrderCancelOperation{
		Owner:   owner,
		OrderID: orderid,
	}

	resp, err := api.Send_Trx(owner, tx)
	if err != nil {
		return errors.Wrapf(err, "Error LimitOrderCancel: ")
	} else {
		log.Println("[LimitOrderCancel] Block -> ", resp.BlockNum, " LimitOrderCancel user -> ", owner)
		return nil
	}
}

func (api *Client) LimitOrderCreate(owner, sell, buy string, orderid uint32) error {

	expiration := time.Now().Add(3600000 * time.Second).UTC()
	fok := false

	tx := &types.LimitOrderCreateOperation{
		Owner:        owner,
		OrderID:      orderid,
		AmountToSell: sell,
		MinToReceive: buy,
		FillOrKill:   fok,
		Expiration:   &types.Time{&expiration},
	}

	resp, err := api.Send_Trx(owner, tx)
	if err != nil {
		return errors.Wrapf(err, "Error LimitOrderCreate: ")
	} else {
		log.Println("[LimitOrderCreate] Block -> ", resp.BlockNum, " LimitOrderCreate user -> ", owner)
		return nil
	}
}

func (api *Client) Convert(owner, amount string, requestid uint32) error {
	tx := &types.ConvertOperation{
		Owner:     owner,
		RequestID: requestid,
		Amount:    amount,
	}

	resp, err := api.Send_Trx(owner, tx)
	if err != nil {
		return errors.Wrapf(err, "Error Convert: ")
	} else {
		log.Println("[Convert] Block -> ", resp.BlockNum, " Convert user -> ", owner)
		return nil
	}
}

func (api *Client) TransferToVesting(from, to, amount string) error {
	tx := &types.TransferToVestingOperation{
		From:   from,
		To:     to,
		Amount: amount,
	}

	resp, err := api.Send_Trx(from, tx)
	if err != nil {
		return errors.Wrapf(err, "Error TransferToVesting: ")
	} else {
		log.Println("[TransferToVesting] Block -> ", resp.BlockNum, " TransferToVesting user -> ", from)
		return nil
	}
}

func (api *Client) WithdrawVesting(account, vshares string) error {
	tx := &types.WithdrawVestingOperation{
		Account:       account,
		VestingShares: vshares,
	}

	resp, err := api.Send_Trx(account, tx)
	if err != nil {
		return errors.Wrapf(err, "Error WithdrawVesting: ")
	} else {
		log.Println("[WithdrawVesting] Block -> ", resp.BlockNum, " WithdrawVesting user -> ", account)
		return nil
	}
}

func (api *Client) ChangeRecoveryAccount(accounttorecover, newrecoveryaccount string) error {
	tx := &types.ChangeRecoveryAccountOperation{
		AccountToRecover:   accounttorecover,
		NewRecoveryAccount: newrecoveryaccount,
		Extensions:         []interface{}{},
	}

	resp, err := api.Send_Trx(accounttorecover, tx)
	if err != nil {
		return errors.Wrapf(err, "Error ChangeRecoveryAccount: ")
	} else {
		log.Println("[ChangeRecoveryAccount] Block -> ", resp.BlockNum, " ChangeRecoveryAccount user -> ", accounttorecover)
		return nil
	}
}

func (api *Client) TransferToSavings(from, to, amount, memo string) error {
	tx := &types.TransferToSavingsOperation{
		From:   from,
		To:     to,
		Amount: amount,
		Memo:   memo,
	}

	resp, err := api.Send_Trx(from, tx)
	if err != nil {
		return errors.Wrapf(err, "Error TransferToSavings: ")
	} else {
		log.Println("[TransferToSavings] Block -> ", resp.BlockNum, " TransferToSavings user -> ", from)
		return nil
	}
}

func (api *Client) TransferFromSavings(from, to, amount, memo string, requestid uint32) error {
	tx := &types.TransferFromSavingsOperation{
		From:      from,
		RequestId: requestid,
		To:        to,
		Amount:    amount,
		Memo:      memo,
	}

	resp, err := api.Send_Trx(from, tx)
	if err != nil {
		return errors.Wrapf(err, "Error TransferFromSavings: ")
	} else {
		log.Println("[TransferFromSavings] Block -> ", resp.BlockNum, " TransferFromSavings user -> ", from)
		return nil
	}
}

func (api *Client) CancelTransferFromSavings(from string, requestid uint32) error {
	tx := &types.CancelTransferFromSavingsOperation{
		From:      from,
		RequestId: requestid,
	}

	resp, err := api.Send_Trx(from, tx)
	if err != nil {
		return errors.Wrapf(err, "Error CancelTransferFromSavings: ")
	} else {
		log.Println("[CancelTransferFromSavings] Block -> ", resp.BlockNum, " CancelTransferFromSavings user -> ", from)
		return nil
	}
}

func (api *Client) DeclineVotingRights(account string, decline bool) error {
	tx := &types.DeclineVotingRightsOperation{
		Account: account,
		Decline: decline,
	}

	resp, err := api.Send_Trx(account, tx)
	if err != nil {
		return errors.Wrapf(err, "Error DeclineVotingRights: ")
	} else {
		log.Println("[DeclineVotingRights] Block -> ", resp.BlockNum, " DeclineVotingRights user -> ", account)
		return nil
	}
}

func (api *Client) FeedPublish(publisher, base, quote string) error {
	tx := &types.FeedPublishOperation{
		Publisher: publisher,
		ExchangeRate: types.ExchRate{
			Base:  base,
			Quote: quote,
		},
	}

	resp, err := api.Send_Trx(publisher, tx)
	if err != nil {
		return errors.Wrapf(err, "Error FeedPublish: ")
	} else {
		log.Println("[FeedPublish] Block -> ", resp.BlockNum, " FeedPublish user -> ", publisher)
		return nil
	}
}

func (api *Client) Comment_Link(user_name, author_name, ppermlink, body string, v *PC_Vote, o *PC_Options) (string, error) {
	var trx []types.Operation

	times, _ := strconv.Unquote(time.Now().Add(30 * time.Second).UTC().Format(fdt))
	permlink := "re-" + author_name + "-" + ppermlink + "-" + times
	permlink = strings.Replace(permlink, ".", "-", -1)

	tx := &types.CommentOperation{
		ParentAuthor:   author_name,
		ParentPermlink: ppermlink,
		Author:         user_name,
		Permlink:       permlink,
		Title:          "",
		Body:           body,
		JsonMetadata:   "{\"app\":\"steem-go\"}",
	}
	trx = append(trx, tx)

	if o != nil {
		symbol := "SBD"
		MAP := "1000000.000 " + symbol
		PSD := o.Percent
		if o.Percent == 0 {
			MAP = "0.000 " + symbol
			PSD = 10000
		} else if o.Percent == 50 {
			PSD = 10000
		} else {
			PSD = 0
		}

		var ext []interface{}
		if len(o.BenefList) > 0 {
			var ben_list []types.Beneficiarie
			var benef types.CommentPayoutBeneficiaries
			for _, val := range o.BenefList {
				ben_list = append(ben_list, types.Beneficiarie{val.Account, val.Weight})
			}
			benef.Beneficiaries = ben_list
			ext = append(ext, 0)
			ext = append(ext, benef)
		}
		Extens := []interface{}{}
		if len(ext) > 0 {
			Extens = []interface{}{ext}
		}

		txo := &types.CommentOptionsOperation{
			Author:               user_name,
			Permlink:             permlink,
			MaxAcceptedPayout:    MAP,
			PercentSteemDollars:  PSD,
			AllowVotes:           true,
			AllowCurationRewards: true,
			Extensions:           Extens,
		}
		trx = append(trx, txo)
	}

	if v != nil && v.Weight != 0 {
		txv := &types.VoteOperation{
			Voter:    user_name,
			Author:   user_name,
			Permlink: permlink,
			Weight:   types.Int16(v.Weight),
		}
		trx = append(trx, txv)
	}

	resp, err := api.Send_Arr_Trx(user_name, trx)
	if err != nil {
		return "", errors.Wrapf(err, "Error Comment : ")
	} else {
		log.Println("[Comment] Block -> ", resp.BlockNum, " User -> ", user_name)
		return permlink, nil
	}
}

func (api *Client) DelegateVestingShares(from, to, share string) error {
	var trx []types.Operation

	tx := &types.DelegateVestingSharesOperation{
		Delegator:     from,
		Delegatee:     to,
		VestingShares: share,
	}
	trx = append(trx, tx)

	resp, err := api.Send_Arr_Trx(from, trx)
	if err != nil {
		return errors.Wrapf(err, "Error DelegateVestingShares: ")
	} else {
		log.Println("[DelegateVestingShares] Block -> ", resp.BlockNum, " DelegateVestingShares user -> ", from)
		return nil
	}
}
