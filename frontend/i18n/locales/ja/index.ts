import common from './common';
import components from './components';
import errors from './errors';
import layout from './layout';
import pages from './pages';
import terms from './terms';

const messages = { common, terms, layout, components, pages, errors };

// 他言語の型検査の基準
export type Messages = typeof messages;

export default messages;
