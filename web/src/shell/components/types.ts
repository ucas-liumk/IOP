export interface Column {
  key: string;
  label: string;
  width?: string | number;
  /**
   * Column alignment. `"tabular"` right-aligns and applies tabular-nums for
   * numeric / date columns so digits line up across rows.
   */
  align?: "left" | "right" | "center" | "tabular";
}
