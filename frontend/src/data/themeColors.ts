import {
  blue,
  cyan,
  geekblue,
  gold,
  green,
  grey,
  lime,
  magenta,
  orange,
  purple,
  red,
  volcano,
  yellow,
} from '@ant-design/colors';

export interface ThemeColor {
  name: string;
  color: string;
  primary: string;
}

export const THEME_COLORS: ThemeColor[] = [
  { name: 'Cyan', color: 'cyan', primary: cyan[5] },
  { name: 'Blue', color: 'blue', primary: blue[5] },
  { name: 'GeekBlue', color: 'geekblue', primary: geekblue[5] },
  { name: 'Purple', color: 'purple', primary: purple[5] },
  { name: 'Magenta', color: 'magenta', primary: magenta[5] },
  { name: 'Red', color: 'red', primary: red[5] },
  { name: 'Volcano', color: 'volcano', primary: volcano[5] },
  { name: 'Orange', color: 'orange', primary: orange[5] },
  { name: 'Gold', color: 'gold', primary: gold[5] },
  { name: 'Yellow', color: 'yellow', primary: yellow[5] },
  { name: 'Lime', color: 'lime', primary: lime[5] },
  { name: 'Green', color: 'green', primary: green[5] },
  { name: 'Grey', color: 'grey', primary: grey[5] },
];

export const DEFAULT_THEME_COLOR = 'cyan';

export const getThemeColor = (name: string): string => {
  const theme = THEME_COLORS.find((c) => c.name.toLowerCase() === name.toLowerCase());
  return theme ? theme.primary : cyan[5];
};
