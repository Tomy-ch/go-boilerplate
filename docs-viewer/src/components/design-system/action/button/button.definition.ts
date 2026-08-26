const DEFAULT_BUTTON_VARIANT = "default";
const OUTLINE_BUTTON_VARIANT = "outline";
const GHOST_BUTTON_VARIANT = "ghost";
const DEFAULT_BUTTON_SIZE = "default";
const SMALL_BUTTON_SIZE = "sm";
const LARGE_BUTTON_SIZE = "lg";

/**
 * Button の操作上の優先度を表す見た目の定数。
 */
export const BUTTON_VARIANT: Readonly<{
  DEFAULT: "default";
  OUTLINE: "outline";
  GHOST: "ghost";
}> = {
  DEFAULT: DEFAULT_BUTTON_VARIANT,
  OUTLINE: OUTLINE_BUTTON_VARIANT,
  GHOST: GHOST_BUTTON_VARIANT,
};

/**
 * Button の高さと横方向の余白を表す大きさの定数。
 */
export const BUTTON_SIZE: Readonly<{
  DEFAULT: "default";
  SMALL: "sm";
  LARGE: "lg";
}> = {
  DEFAULT: DEFAULT_BUTTON_SIZE,
  SMALL: SMALL_BUTTON_SIZE,
  LARGE: LARGE_BUTTON_SIZE,
};

/** {@link BUTTON_VARIANT} のいずれか。 */
export type ButtonVariant = (typeof BUTTON_VARIANT)[keyof typeof BUTTON_VARIANT];

/** {@link BUTTON_SIZE} のいずれか。 */
export type ButtonSize = (typeof BUTTON_SIZE)[keyof typeof BUTTON_SIZE];
