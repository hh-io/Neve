import { ref } from 'vue';

// 主题切换计数器:ECharts 的 canvas 不解析 CSS 变量,图表 option 里的颜色
// 必须用 getThemeColor 取实际值;在 computed 中读取 themeVersion.value
// 可以让主题切换时 option 重新计算、图表重新取色。
export const themeVersion = ref(0);

export function bumpThemeVersion(): void {
    themeVersion.value++;
}

export function getThemeColor(variableName: string): string {
    const style = getComputedStyle(document.documentElement);
    return style.getPropertyValue(variableName).trim();
}
