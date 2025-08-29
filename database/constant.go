package database

type TxStatus string

// 提现是没有确认位的
const (
	//====================父交易的状态==========================
	TxStatusWaitSign             TxStatus = "wait_sign"           // 交易等待签名
	TxStatusUnSent               TxStatus = "unsend"              // 交易未发送
	TxStatusSent                 TxStatus = "sent"                // 交易已发送
	TxStatusSentNotify           TxStatus = "sent_notify_success" // 交易以广播通知
	TxStatusSentNotifyFail       TxStatus = "sent_notify_fail"    // 交易以广播失败通知
	TxStatusWithdrawed           TxStatus = "withdrawed"          // 交易已发送
	TxStatusWithdrawedNotify     TxStatus = "withdrawed_notify_success"
	TxStatusWithdrawedNotifyFail TxStatus = "withdrawed_notify_fail"

	TxStatusUnSafe              TxStatus = "unsafe"                   // 链上扫到交易
	TxStatusSafe                TxStatus = "safe"                     // 交易过了安全确认位
	TxStatusFinalized           TxStatus = "finalized"                // 交易已完成，可以提现
	TxStatusUnSafeNotify        TxStatus = "unsafe_notify_success"    // 链上扫到交易已通知
	TxStatusSafeNotify          TxStatus = "safe_notify_success"      // 交易过了安全确认位已通知
	TxStatusFinalizedNotify     TxStatus = "finalized_notify_success" // 交易完成已通知
	TxStatusUnSafeNotifyFail    TxStatus = "unsafe_notify_fail"       // 链上扫到交易通知失败
	TxStatusSafeNotifyFail      TxStatus = "safe_notify_fail"         // 交易过了安全确认位通知失败
	TxStatusFinalizedNotifyFail TxStatus = "finalized_notify_fail"    // 交易完成通知失败

	TxStatusSuccess        TxStatus = "done_success"
	TxStatusFail           TxStatus = "done_fail"
	TxStatusFailNotify     TxStatus = "done_fail_notify_success"
	TxStatusFailNotifyFail TxStatus = "done_fail_notify_fail"

	TxStatusFallback           TxStatus = "fallback"                // 交易回滚状态
	TxStatusFallbackNotify     TxStatus = "fallback_notify_success" // 交易回滚通知成功
	TxStatusFallbackNotifyFail TxStatus = "fallback_notify_fail"    // 交易回滚通知失败
	TxStatusFallbackDone       TxStatus = "done_fallback"           // 交易回滚状态

	TxStatusInternalCallBack TxStatus = "send_to_business_for_sign"

	//====================子交易的状体==========================

)
