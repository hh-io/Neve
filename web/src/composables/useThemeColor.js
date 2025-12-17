export function getThemeColor(variableName) {
    const style = getComputedStyle(document.documentElement);
    return style.getPropertyValue(variableName).trim();
}
