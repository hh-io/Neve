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

export function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return `${date.getMonth() + 1}/${date.getDate()}`;
}

export function formatDateTime(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleString("zh-CN");
}

export function formatAccountName(account: string): string {
    const parts = account.split(":");
    return parts.slice(-2).join(" · ");
}

export function getTagClass(tag: string): string {
    const tagLower = tag.toLowerCase();
    if (tagLower.includes('travel') || tagLower.includes('旅行')) return 'tag-travel';
    if (tagLower.includes('food') || tagLower.includes('餐饮')) return 'tag-food';
    if (tagLower.includes('work') || tagLower.includes('工作')) return 'tag-work';
    return 'tag-default';
}
