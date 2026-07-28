// Shared formatting utilities
export function formatMoney(amount: number | null | undefined): string {
    if (amount == null) return "¥0.00";
    const sign = amount < 0 ? "-" : "";
    const absAmount = Math.abs(amount);
    return (
        sign +
        "¥" +
        absAmount.toLocaleString("zh-CN", {
            minimumFractionDigits: 2,
            maximumFractionDigits: 2,
        })
    );
}

export function formatDateTime(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleString("zh-CN");
}

