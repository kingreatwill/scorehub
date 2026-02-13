export type BankMeta = {
  name: string
  abbr: string
  logo: string
  wordmark: string
}

function normalizeBase(v: string): string {
  const s = String(v || '').trim()
  return s.replace(/\/+$/, '')
}

function resolveAssetBase(): string {
  const direct = String(import.meta.env.VITE_SCOREHUB_ASSET_BASE || '').trim()
  if (direct) return normalizeBase(direct)
  const api = normalizeBase(import.meta.env.VITE_SCOREHUB_API_BASE || 'https://wxapi.wcoder.com/api/v1')
  return api.replace(/\/api\/v1\/?$/, '')
}

const ASSET_BASE = resolveAssetBase()

function withAssetBase(path: string): string {
  const raw = String(path || '').trim()
  if (!raw) return ''
  if (/^https?:\/\//i.test(raw)) return raw
  const cleaned = raw.startsWith('/') ? raw : `/${raw}`
  return `${ASSET_BASE}${cleaned}`
}

const rawBankList: BankMeta[] = [
  {
    name: '中国工商银行',
    abbr: 'ICBC',
    logo: '/static/img/pay/ICBC.svg',
    wordmark: '/static/img/pay/ICBC_wordmark.svg',
  },
  {
    name: '中国农业银行',
    abbr: 'ABC',
    logo: '/static/img/pay/ABC.svg',
    wordmark: '/static/img/pay/ABC_wordmark.svg',
  },
  {
    name: '中国银行',
    abbr: 'BOC',
    logo: '/static/img/pay/BOC.svg',
    wordmark: '/static/img/pay/BOC_wordmark.svg',
  },
  {
    name: '中国建设银行',
    abbr: 'CCB',
    logo: '/static/img/pay/CCB.svg',
    wordmark: '/static/img/pay/CCB_wordmark.svg',
  },
  {
    name: '交通银行',
    abbr: 'COMM',
    logo: '/static/img/pay/COMM.svg',
    wordmark: '/static/img/pay/COMM_wordmark.svg',
  },
  {
    name: '中国邮政储蓄银行',
    abbr: 'PSBC',
    logo: '/static/img/pay/PSBC.svg',
    wordmark: '/static/img/pay/PSBC_wordmark.svg',
  },
  {
    name: '招商银行',
    abbr: 'CMB',
    logo: '/static/img/pay/CMB.svg',
    wordmark: '/static/img/pay/CMB_wordmark.svg',
  },
  {
    name: '中信银行',
    abbr: 'CITIC',
    logo: '/static/img/pay/CITIC.svg',
    wordmark: '/static/img/pay/CITIC_wordmark.svg',
  },
  {
    name: '中国光大银行',
    abbr: 'CEB',
    logo: '/static/img/pay/CEB.svg',
    wordmark: '/static/img/pay/CEB_wordmark.svg',
  },
  {
    name: '中国民生银行',
    abbr: 'CMBC',
    logo: '/static/img/pay/CMBC.svg',
    wordmark: '/static/img/pay/CMBC_wordmark.svg',
  },
  {
    name: '浦发银行',
    abbr: 'SPDB',
    logo: '/static/img/pay/SPDB.svg',
    wordmark: '/static/img/pay/SPDB_wordmark.svg',
  },
  {
    name: '兴业银行',
    abbr: 'CIB',
    logo: '/static/img/pay/CIB.svg',
    wordmark: '/static/img/pay/CIB_wordmark.svg',
  },
  {
    name: '平安银行',
    abbr: 'SPABANK',
    logo: '/static/img/pay/SPABANK.svg',
    wordmark: '/static/img/pay/SPABANK_wordmark.svg',
  },
  {
    name: '花呗',
    abbr: 'huabei',
    logo: '/static/img/pay/huabei.svg',
    wordmark: '/static/img/pay/huabei_wordmark.svg',
  },
  {
    name: '微信支付',
    abbr: 'weixinpay',
    logo: '/static/img/pay/weixinpay.svg',
    wordmark: '',
  },
  {
    name: '网商银行',
    abbr: 'Mybank',
    logo: '/static/img/pay/Mybank.svg',
    wordmark: '/static/img/pay/Mybank_wordmark.svg',
  },
  {
    name: '芝麻信用',
    abbr: 'zhimaxinyong',
    logo: '/static/img/pay/zhimaxinyong.svg',
    wordmark: '/static/img/pay/zhimaxinyong_wordmark.svg',
  },
  {
    name: '余额宝',
    abbr: 'yuebao',
    logo: '/static/img/pay/yuebao.svg',
    wordmark: '/static/img/pay/yuebao_wordmark.svg',
  },
  {
    name: '借呗',
    abbr: 'jiebei',
    logo: '/static/img/pay/jiebei.svg',
    wordmark: '/static/img/pay/jiebei_wordmark.svg',
  },
  {
    name: '西安银行',
    abbr: 'XABANK',
    logo: '/static/img/pay/XABANK.svg',
    wordmark: '/static/img/pay/XABANK_wordmark.svg',
  },
  {
    name: '桂林银行',
    abbr: 'GLBANK',
    logo: '/static/img/pay/GLBANK.svg',
    wordmark: '/static/img/pay/GLBANK_wordmark.svg',
  },
  {
    name: '泰安银行',
    abbr: 'TACCB',
    logo: '/static/img/pay/TACCB.svg',
    wordmark: '/static/img/pay/TACCB_wordmark.svg',
  },
  {
    name: '浙江稠州商业银行',
    abbr: 'CZCB',
    logo: '/static/img/pay/CZCB.svg',
    wordmark: '/static/img/pay/CZCB_wordmark.svg',
  },
  {
    name: '营口沿海银行',
    abbr: 'YKYHB',
    logo: '/static/img/pay/YKYHB.svg',
    wordmark: '',
  },
  {
    name: '晋中银行',
    abbr: 'JZB',
    logo: '/static/img/pay/JZB.svg',
    wordmark: '',
  },
  {
    name: '青海银行',
    abbr: 'QHBANK',
    logo: '/static/img/pay/QHBANK.svg',
    wordmark: '/static/img/pay/QHBANK_wordmark.svg',
  },
  {
    name: '雅安市商业银行',
    abbr: 'YACCB',
    logo: '/static/img/pay/YACCB.svg',
    wordmark: '/static/img/pay/YACCB_wordmark.svg',
  },
  {
    name: '广州银行',
    abbr: 'GZCB',
    logo: '/static/img/pay/GZCB.svg',
    wordmark: '/static/img/pay/GZCB_wordmark.svg',
  },
  {
    name: '德州银行',
    abbr: 'DZBANK',
    logo: '/static/img/pay/DZBANK.svg',
    wordmark: '/static/img/pay/DZBANK_wordmark.svg',
  },
  {
    name: '湖州银行',
    abbr: 'BOHZ',
    logo: '/static/img/pay/BOHZ.svg',
    wordmark: '/static/img/pay/BOHZ_wordmark.svg',
  },
  {
    name: '抚顺银行',
    abbr: 'FSCB',
    logo: '/static/img/pay/FSCB.svg',
    wordmark: '/static/img/pay/FSCB_wordmark.svg',
  },
  {
    name: '唐山银行',
    abbr: 'BOTS',
    logo: '/static/img/pay/BOTS.svg',
    wordmark: '/static/img/pay/BOTS_wordmark.svg',
  },
  {
    name: '昆仑银行',
    abbr: 'KLB',
    logo: '/static/img/pay/KLB.svg',
    wordmark: '/static/img/pay/KLB_wordmark.svg',
  },
  {
    name: '长城华西银行',
    abbr: 'GWB',
    logo: '/static/img/pay/GWB.svg',
    wordmark: '/static/img/pay/GWB_wordmark.svg',
  },
  {
    name: '江西银行',
    abbr: 'JXB',
    logo: '/static/img/pay/JXB.svg',
    wordmark: '/static/img/pay/JXB_wordmark.svg',
  },
  {
    name: '苏州银行',
    abbr: 'BOSZ',
    logo: '/static/img/pay/BOSZ.svg',
    wordmark: '/static/img/pay/BOSZ_wordmark.svg',
  },
  {
    name: '大连银行',
    abbr: 'DLB',
    logo: '/static/img/pay/DLB.svg',
    wordmark: '/static/img/pay/DLB_wordmark.svg',
  },
  {
    name: '沧州银行',
    abbr: 'BOCZ',
    logo: '/static/img/pay/BOCZ.svg',
    wordmark: '/static/img/pay/BOCZ_wordmark.svg',
  },
  {
    name: '长安银行',
    abbr: 'CABANK',
    logo: '/static/img/pay/CABANK.svg',
    wordmark: '/static/img/pay/CABANK_wordmark.svg',
  },
  {
    name: '柳州银行',
    abbr: 'LZCCB',
    logo: '/static/img/pay/LZCCB.svg',
    wordmark: '/static/img/pay/LZCCB_wordmark.svg',
  },
  {
    name: '威海市商业银行',
    abbr: 'WHCCB',
    logo: '/static/img/pay/WHCCB.svg',
    wordmark: '/static/img/pay/WHCCB_wordmark.svg',
  },
  {
    name: '浙江民泰商业银行',
    abbr: 'MTBANK',
    logo: '/static/img/pay/MTBANK.svg',
    wordmark: '/static/img/pay/MTBANK_wordmark.svg',
  },
  {
    name: '营口银行',
    abbr: 'BOYK',
    logo: '/static/img/pay/BOYK.svg',
    wordmark: '/static/img/pay/BOYK_wordmark.svg',
  },
  {
    name: '阳泉市商业银行',
    abbr: 'YQCCB',
    logo: '/static/img/pay/YQCCB.svg',
    wordmark: '/static/img/pay/YQCCB_wordmark.svg',
  },
  {
    name: '西藏银行',
    abbr: 'BOXZ',
    logo: '/static/img/pay/BOXZ.svg',
    wordmark: '/static/img/pay/BOXZ_wordmark.svg',
  },
  {
    name: '宜宾市商业银行',
    abbr: 'YBCCB',
    logo: '/static/img/pay/YBCCB.svg',
    wordmark: '/static/img/pay/YBCCB_wordmark.svg',
  },
  {
    name: '东莞银行',
    abbr: 'BOD',
    logo: '/static/img/pay/BOD.svg',
    wordmark: '/static/img/pay/BOD_wordmark.svg',
  },
  {
    name: '东营银行',
    abbr: 'DYCCB',
    logo: '/static/img/pay/DYCCB.svg',
    wordmark: '/static/img/pay/DYCCB_wordmark.svg',
  },
  {
    name: '嘉兴银行',
    abbr: 'JXBANK',
    logo: '/static/img/pay/JXBANK.svg',
    wordmark: '/static/img/pay/JXBANK_wordmark.svg',
  },
  {
    name: '阜阳银行',
    abbr: 'FXCB',
    logo: '/static/img/pay/FXCB.svg',
    wordmark: '/static/img/pay/FXCB_wordmark.svg',
  },
  {
    name: '邢台银行',
    abbr: 'XTB',
    logo: '/static/img/pay/XTB.svg',
    wordmark: '/static/img/pay/XTB_wordmark.svg',
  },
  {
    name: '哈密市商业银行',
    abbr: 'HMCCB',
    logo: '/static/img/pay/HMCCB.svg',
    wordmark: '/static/img/pay/HMCCB_wordmark.svg',
  },
  {
    name: '乐山市商业银行',
    abbr: 'LSCCB',
    logo: '/static/img/pay/LSCCB.svg',
    wordmark: '/static/img/pay/LSCCB_wordmark.svg',
  },
  {
    name: '焦作中旅银行',
    abbr: 'CTS',
    logo: '/static/img/pay/CTS.svg',
    wordmark: '/static/img/pay/CTS_wordmark.svg',
  },
  {
    name: '九江银行',
    abbr: 'JJCCB',
    logo: '/static/img/pay/JJCCB.svg',
    wordmark: '/static/img/pay/JJCCB_wordmark.svg',
  },
  {
    name: '江苏长江商业银行',
    abbr: 'JSCJCB',
    logo: '/static/img/pay/JSCJCB.svg',
    wordmark: '/static/img/pay/JSCJCB_wordmark.svg',
  },
  {
    name: '锦州银行',
    abbr: 'JZBANK',
    logo: '/static/img/pay/JZBANK.svg',
    wordmark: '/static/img/pay/JZBANK_wordmark.svg',
  },
  {
    name: '承德银行',
    abbr: 'CDBANK',
    logo: '/static/img/pay/CDBANK.svg',
    wordmark: '/static/img/pay/CDBANK_wordmark.svg',
  },
  {
    name: '云南红塔银行',
    abbr: 'YNHTBANK',
    logo: '/static/img/pay/YNHTBANK.svg',
    wordmark: '/static/img/pay/YNHTBANK_wordmark.svg',
  },
  {
    name: '广西北部湾银行',
    abbr: 'BGB',
    logo: '/static/img/pay/BGB.svg',
    wordmark: '/static/img/pay/BGB_wordmark.svg',
  },
  {
    name: '日照银行',
    abbr: 'RZB',
    logo: '/static/img/pay/RZB.svg',
    wordmark: '/static/img/pay/RZB_wordmark.svg',
  },
  {
    name: '温州银行',
    abbr: 'WZBANK',
    logo: '/static/img/pay/WZBANK.svg',
    wordmark: '/static/img/pay/WZBANK_wordmark.svg',
  },
  {
    name: '铁岭银行',
    abbr: 'BOTL',
    logo: '/static/img/pay/BOTL.svg',
    wordmark: '/static/img/pay/BOTL_wordmark.svg',
  },
  {
    name: '晋城银行',
    abbr: 'JINCHB',
    logo: '/static/img/pay/JINCHB.svg',
    wordmark: '/static/img/pay/JINCHB_wordmark.svg',
  },
  {
    name: '石嘴山银行',
    abbr: 'SZSBK',
    logo: '/static/img/pay/SZSBK.svg',
    wordmark: '/static/img/pay/SZSBK_wordmark.svg',
  },
  {
    name: '遂宁银行',
    abbr: 'SNBANK',
    logo: '/static/img/pay/SNBANK.svg',
    wordmark: '/static/img/pay/SNBANK_wordmark.svg',
  },
  {
    name: '湖北银行',
    abbr: 'HBC',
    logo: '/static/img/pay/HBC.svg',
    wordmark: '/static/img/pay/HBC_wordmark.svg',
  },
  {
    name: '青岛银行',
    abbr: 'QDCCB',
    logo: '/static/img/pay/QDCCB.svg',
    wordmark: '/static/img/pay/QDCCB_wordmark.svg',
  },
  {
    name: '宁波东海银行',
    abbr: 'NDHB',
    logo: '/static/img/pay/NDHB.svg',
    wordmark: '/static/img/pay/NDHB_wordmark.svg',
  },
  {
    name: '丹东银行',
    abbr: 'BODD',
    logo: '/static/img/pay/BODD.svg',
    wordmark: '/static/img/pay/BODD_wordmark.svg',
  },
  {
    name: '秦皇岛银行',
    abbr: 'QHDBANK',
    logo: '/static/img/pay/QHDBANK.svg',
    wordmark: '/static/img/pay/QHDBANK_wordmark.svg',
  },
  {
    name: '乌鲁木齐市商业银行',
    abbr: 'UCCB',
    logo: '/static/img/pay/UCCB.svg',
    wordmark: '/static/img/pay/UCCB_wordmark.svg',
  },
  {
    name: '达州银行',
    abbr: 'DCCB',
    logo: '/static/img/pay/DCCB.svg',
    wordmark: '/static/img/pay/DCCB_wordmark.svg',
  },
  {
    name: '平顶山银行',
    abbr: 'BOP',
    logo: '/static/img/pay/BOP.svg',
    wordmark: '/static/img/pay/BOP_wordmark.svg',
  },
  {
    name: '厦门国际银行',
    abbr: 'XMINTB',
    logo: '/static/img/pay/XMINTB.svg',
    wordmark: '/static/img/pay/XMINTB_wordmark.svg',
  },
  {
    name: '南京银行',
    abbr: 'NJCB',
    logo: '/static/img/pay/NJCB.svg',
    wordmark: '/static/img/pay/NJCB_wordmark.svg',
  },
  {
    name: '盛京银行',
    abbr: 'SJBANK',
    logo: '/static/img/pay/SJBANK.svg',
    wordmark: '/static/img/pay/SJBANK_wordmark.svg',
  },
  {
    name: '保定银行',
    abbr: 'BOBD',
    logo: '/static/img/pay/BOBD.svg',
    wordmark: '/static/img/pay/BOBD_wordmark.svg',
  },
  {
    name: '曲靖市商业银行',
    abbr: 'QJCCCB',
    logo: '/static/img/pay/QJCCCB.svg',
    wordmark: '/static/img/pay/QJCCCB_wordmark.svg',
  },
  {
    name: '珠海华润银行',
    abbr: 'RBOZ',
    logo: '/static/img/pay/RBOZ.svg',
    wordmark: '/static/img/pay/RBOZ_wordmark.svg',
  },
  {
    name: '齐商银行',
    abbr: 'QSB',
    logo: '/static/img/pay/QSB.svg',
    wordmark: '/static/img/pay/QSB_wordmark.svg',
  },
  {
    name: '台州银行',
    abbr: 'TZBANK',
    logo: '/static/img/pay/TZBANK.svg',
    wordmark: '/static/img/pay/TZBANK_wordmark.svg',
  },
  {
    name: '盘锦银行',
    abbr: 'BOPJ',
    logo: '/static/img/pay/BOPJ.svg',
    wordmark: '/static/img/pay/BOPJ_wordmark.svg',
  },
  {
    name: '大同银行',
    abbr: 'DTB',
    logo: '/static/img/pay/DTB.svg',
    wordmark: '/static/img/pay/DTB_wordmark.svg',
  },
  {
    name: '宁夏银行',
    abbr: 'NXBANK',
    logo: '/static/img/pay/NXBANK.svg',
    wordmark: '/static/img/pay/NXBANK_wordmark.svg',
  },
  {
    name: '四川天府银行',
    abbr: 'SCTFB',
    logo: '/static/img/pay/SCTFB.svg',
    wordmark: '/static/img/pay/SCTFB_wordmark.svg',
  },
  {
    name: '汉口银行',
    abbr: 'HKB',
    logo: '/static/img/pay/HKB.svg',
    wordmark: '/static/img/pay/HKB_wordmark.svg',
  },
  {
    name: '莱商银行',
    abbr: 'LSBANK',
    logo: '/static/img/pay/LSBANK.svg',
    wordmark: '/static/img/pay/LSBANK_wordmark.svg',
  },
  {
    name: '宁波通商银行',
    abbr: 'NBCMB',
    logo: '/static/img/pay/NBCMB.svg',
    wordmark: '/static/img/pay/NBCMB_wordmark.svg',
  },
  {
    name: '朝阳银行',
    abbr: 'BOCY',
    logo: '/static/img/pay/BOCY.svg',
    wordmark: '/static/img/pay/BOCY_wordmark.svg',
  },
  {
    name: '廊坊银行',
    abbr: 'BOLF',
    logo: '/static/img/pay/BOLF.svg',
    wordmark: '/static/img/pay/BOLF_wordmark.svg',
  },
  {
    name: '新疆银行',
    abbr: 'XJB',
    logo: '/static/img/pay/XJB.svg',
    wordmark: '/static/img/pay/XJB_wordmark.svg',
  },
  {
    name: '成都银行',
    abbr: 'CDCB',
    logo: '/static/img/pay/CDCB.svg',
    wordmark: '/static/img/pay/CDCB_wordmark.svg',
  },
  {
    name: '郑州银行',
    abbr: 'ZZBANK',
    logo: '/static/img/pay/ZZBANK.svg',
    wordmark: '/static/img/pay/ZZBANK_wordmark.svg',
  },
  {
    name: '厦门银行',
    abbr: 'XMBANK',
    logo: '/static/img/pay/XMBANK.svg',
    wordmark: '/static/img/pay/XMBANK_wordmark.svg',
  },
  {
    name: '江苏银行',
    abbr: 'JSBANK',
    logo: '/static/img/pay/JSBANK.svg',
    wordmark: '/static/img/pay/JSBANK_wordmark.svg',
  },
  {
    name: '乌海银行',
    abbr: 'WHBANK',
    logo: '/static/img/pay/WHBANK.svg',
    wordmark: '/static/img/pay/WHBANK_wordmark.svg',
  },
  {
    name: '河北银行',
    abbr: 'BHB',
    logo: '/static/img/pay/BHB.svg',
    wordmark: '/static/img/pay/BHB_wordmark.svg',
  },
  {
    name: '兰州银行',
    abbr: 'LZBANK',
    logo: '/static/img/pay/LZBANK.svg',
    wordmark: '/static/img/pay/LZBANK_wordmark.svg',
  },
  {
    name: '重庆三峡银行',
    abbr: 'CCQTGB',
    logo: '/static/img/pay/CCQTGB.svg',
    wordmark: '/static/img/pay/CCQTGB_wordmark.svg',
  },
  {
    name: '烟台银行',
    abbr: 'YTB',
    logo: '/static/img/pay/YTB.svg',
    wordmark: '/static/img/pay/YTB_wordmark.svg',
  },
  {
    name: '徽商银行',
    abbr: 'HSBANK',
    logo: '/static/img/pay/HSBANK.svg',
    wordmark: '/static/img/pay/HSBANK_wordmark.svg',
  },
  {
    name: '哈尔滨银行',
    abbr: 'HRBCB',
    logo: '/static/img/pay/HRBCB.svg',
    wordmark: '/static/img/pay/HRBCB_wordmark.svg',
  },
  {
    name: '富滇银行',
    abbr: 'FDBANK',
    logo: '/static/img/pay/FDBANK.svg',
    wordmark: '/static/img/pay/FDBANK_wordmark.svg',
  },
  {
    name: '广东南粤银行',
    abbr: 'NYBANK',
    logo: '/static/img/pay/NYBANK.svg',
    wordmark: '/static/img/pay/NYBANK_wordmark.svg',
  },
  {
    name: '临商银行',
    abbr: 'LSBC',
    logo: '/static/img/pay/LSBC.svg',
    wordmark: '/static/img/pay/LSBC_wordmark.svg',
  },
  {
    name: '绍兴银行',
    abbr: 'SXCB',
    logo: '/static/img/pay/SXCB.svg',
    wordmark: '/static/img/pay/SXCB_wordmark.svg',
  },
  {
    name: '辽阳银行',
    abbr: 'BOLY',
    logo: '/static/img/pay/BOLY.svg',
    wordmark: '/static/img/pay/BOLY_wordmark.svg',
  },
  {
    name: '晋商银行',
    abbr: 'JSB',
    logo: '/static/img/pay/JSB.svg',
    wordmark: '/static/img/pay/JSB_wordmark.svg',
  },
  {
    name: '新疆汇和银行',
    abbr: 'XJHB',
    logo: '/static/img/pay/XJHB.svg',
    wordmark: '/static/img/pay/XJHB_wordmark.svg',
  },
  {
    name: '绵阳市商业银行',
    abbr: 'MYCCB',
    logo: '/static/img/pay/MYCCB.svg',
    wordmark: '/static/img/pay/MYCCB_wordmark.svg',
  },
  {
    name: '长沙银行',
    abbr: 'BSCB',
    logo: '/static/img/pay/BSCB.svg',
    wordmark: '/static/img/pay/BSCB_wordmark.svg',
  },
  {
    name: '齐鲁银行',
    abbr: 'QLBANK',
    logo: '/static/img/pay/QLBANK.svg',
    wordmark: '/static/img/pay/QLBANK_wordmark.svg',
  },
  {
    name: '宁波银行',
    abbr: 'NBCB',
    logo: '/static/img/pay/NBCB.svg',
    wordmark: '/static/img/pay/NBCB_wordmark.svg',
  },
  {
    name: '本溪市商业银行',
    abbr: 'BCCB',
    logo: '/static/img/pay/BCCB.svg',
    wordmark: '/static/img/pay/BCCB_wordmark.svg',
  },
  {
    name: '衡水银行',
    abbr: 'BOHS',
    logo: '/static/img/pay/BOHS.svg',
    wordmark: '/static/img/pay/BOHS_wordmark.svg',
  },
  {
    name: '贵州银行',
    abbr: 'BOGZ',
    logo: '/static/img/pay/BOGZ.svg',
    wordmark: '/static/img/pay/BOGZ_wordmark.svg',
  },
  {
    name: '四川银行',
    abbr: 'SCB',
    logo: '/static/img/pay/SCB.svg',
    wordmark: '/static/img/pay/SCB_wordmark.svg',
  },
  {
    name: '中原银行',
    abbr: 'ZYBANK',
    logo: '/static/img/pay/ZYBANK.svg',
    wordmark: '/static/img/pay/ZYBANK_wordmark.svg',
  },
  {
    name: '泉州银行',
    abbr: 'BOQZ',
    logo: '/static/img/pay/BOQZ.svg',
    wordmark: '/static/img/pay/BOQZ_wordmark.svg',
  },
  {
    name: '上海银行',
    abbr: 'BOSC',
    logo: '/static/img/pay/BOSC.svg',
    wordmark: '/static/img/pay/BOSC_wordmark.svg',
  },
  {
    name: '鄂尔多斯银行',
    abbr: 'ORDOSB',
    logo: '/static/img/pay/ORDOSB.svg',
    wordmark: '/static/img/pay/ORDOSB_wordmark.svg',
  },
  {
    name: '天津银行',
    abbr: 'BOTJ',
    logo: '/static/img/pay/BOTJ.svg',
    wordmark: '/static/img/pay/BOTJ_wordmark.svg',
  },
  {
    name: '甘肃银行',
    abbr: 'BOGS',
    logo: '/static/img/pay/BOGS.svg',
    wordmark: '/static/img/pay/BOGS_wordmark.svg',
  },
  {
    name: '海南银行',
    abbr: 'HNB',
    logo: '/static/img/pay/HNB.svg',
    wordmark: '/static/img/pay/HNB_wordmark.svg',
  },
  {
    name: '潍坊银行',
    abbr: 'WFCCB',
    logo: '/static/img/pay/WFCCB.svg',
    wordmark: '/static/img/pay/WFCCB_wordmark.svg',
  },
  {
    name: '泰隆银行',
    abbr: 'TLCB',
    logo: '/static/img/pay/TLCB.svg',
    wordmark: '/static/img/pay/TLCB_wordmark.svg',
  },
  {
    name: '吉林银行',
    abbr: 'BOJL',
    logo: '/static/img/pay/BOJL.svg',
    wordmark: '/static/img/pay/BOJL_wordmark.svg',
  },
  {
    name: '长治银行',
    abbr: 'CZB',
    logo: '/static/img/pay/CZB.svg',
    wordmark: '/static/img/pay/CZB_wordmark.svg',
  },
  {
    name: '自贡银行',
    abbr: 'ZGBANK',
    logo: '/static/img/pay/ZGBANK.svg',
    wordmark: '/static/img/pay/ZGBANK_wordmark.svg',
  },
  {
    name: '广东华兴银行',
    abbr: 'GHB',
    logo: '/static/img/pay/GHB.svg',
    wordmark: '/static/img/pay/GHB_wordmark.svg',
  },
  {
    name: '济宁银行',
    abbr: 'JNBANK',
    logo: '/static/img/pay/JNBANK.svg',
    wordmark: '/static/img/pay/JNBANK_wordmark.svg',
  },
  {
    name: '金华银行',
    abbr: 'JHCCB',
    logo: '/static/img/pay/JHCCB.svg',
    wordmark: '/static/img/pay/JHCCB_wordmark.svg',
  },
  {
    name: '葫芦岛银行',
    abbr: 'BOHLD',
    logo: '/static/img/pay/BOHLD.svg',
    wordmark: '/static/img/pay/BOHLD_wordmark.svg',
  },
  {
    name: '张家口银行',
    abbr: 'ZJKCCB',
    logo: '/static/img/pay/ZJKCCB.svg',
    wordmark: '/static/img/pay/ZJKCCB_wordmark.svg',
  },
  {
    name: '库尔勒市商业银行',
    abbr: 'KCCCB',
    logo: '/static/img/pay/KCCCB.svg',
    wordmark: '/static/img/pay/KCCCB_wordmark.svg',
  },
  {
    name: '泸州银行',
    abbr: 'LZB',
    logo: '/static/img/pay/LZB.svg',
    wordmark: '/static/img/pay/LZB_wordmark.svg',
  },
  {
    name: '华融湘江银行',
    abbr: 'HRXJB',
    logo: '/static/img/pay/HRXJB.svg',
    wordmark: '/static/img/pay/HRXJB_wordmark.svg',
  },
  {
    name: '上饶银行',
    abbr: 'SRBANK',
    logo: '/static/img/pay/SRBANK.svg',
    wordmark: '/static/img/pay/SRBANK_wordmark.svg',
  },
  {
    name: '杭州银行',
    abbr: 'HZCB',
    logo: '/static/img/pay/HZCB.svg',
    wordmark: '/static/img/pay/HZCB_wordmark.svg',
  },
  {
    name: '鞍山银行',
    abbr: 'BOAS',
    logo: '/static/img/pay/BOAS.svg',
    wordmark: '/static/img/pay/BOAS_wordmark.svg',
  },
  {
    name: '邯郸银行',
    abbr: 'HDBANK',
    logo: '/static/img/pay/HDBANK.svg',
    wordmark: '/static/img/pay/HDBANK_wordmark.svg',
  },
  {
    name: '贵阳银行',
    abbr: 'GYCCB',
    logo: '/static/img/pay/GYCCB.svg',
    wordmark: '/static/img/pay/GYCCB_wordmark.svg',
  },
  {
    name: '重庆银行',
    abbr: 'CQBANK',
    logo: '/static/img/pay/CQBANK.svg',
    wordmark: '/static/img/pay/CQBANK_wordmark.svg',
  },
  {
    name: '枣庄银行',
    abbr: 'ZZB',
    logo: '/static/img/pay/ZZB.svg',
    wordmark: '/static/img/pay/ZZB_wordmark.svg',
  },
  {
    name: '福建海峡银行',
    abbr: 'FJHXBC',
    logo: '/static/img/pay/FJHXBC.svg',
    wordmark: '/static/img/pay/FJHXBC_wordmark.svg',
  },
  {
    name: '龙江银行',
    abbr: 'LJBANK',
    logo: '/static/img/pay/LJBANK.svg',
    wordmark: '/static/img/pay/LJBANK_wordmark.svg',
  },
  {
    name: '内蒙古银行',
    abbr: 'H3CB',
    logo: '/static/img/pay/H3CB.svg',
    wordmark: '/static/img/pay/H3CB_wordmark.svg',
  },
  {
    name: '北京银行',
    abbr: 'BOB',
    logo: '/static/img/pay/BOB.svg',
    wordmark: '/static/img/pay/BOB_wordmark.svg',
  },
  {
    name: '恒丰银行',
    abbr: 'EGBANK',
    logo: '/static/img/pay/EGBANK.svg',
    wordmark: '/static/img/pay/EGBANK_wordmark.svg',
  },
  {
    name: '浙商银行',
    abbr: 'CZBANK',
    logo: '/static/img/pay/CZBANK.svg',
    wordmark: '/static/img/pay/CZBANK_wordmark.svg',
  },
  {
    name: '广发银行',
    abbr: 'GDB',
    logo: '/static/img/pay/GDB.svg',
    wordmark: '/static/img/pay/GDB_wordmark.svg',
  },
  {
    name: '渤海银行',
    abbr: 'BOHAIB',
    logo: '/static/img/pay/BOHAIB.svg',
    wordmark: '/static/img/pay/BOHAIB_wordmark.svg',
  },
  {
    name: '华夏银行',
    abbr: 'HXB',
    logo: '/static/img/pay/HXB.svg',
    wordmark: '/static/img/pay/HXB_wordmark.svg',
  },
  {
    name: '中国进出口银行',
    abbr: 'EIBOF',
    logo: '/static/img/pay/EIBOF.svg',
    wordmark: '/static/img/pay/EIBOF_wordmark.svg',
  },
  {
    name: '国家开发银行',
    abbr: 'CDB',
    logo: '/static/img/pay/CDB.svg',
    wordmark: '/static/img/pay/CDB_wordmark.svg',
  },
  {
    name: '中国人民银行',
    abbr: 'PBOC',
    logo: '/static/img/pay/PBOC.svg',
    wordmark: '/static/img/pay/PBOC_wordmark.svg',
  },
  {
    name: '中国农业发展银行',
    abbr: 'ADBC',
    logo: '/static/img/pay/ADBC.svg',
    wordmark: '',
  },
]

export const bankList: BankMeta[] = rawBankList.map((bank) => ({
  ...bank,
  logo: withAssetBase(bank.logo),
  wordmark: withAssetBase(bank.wordmark),
}))

export function getBankMeta(name: string): BankMeta | null {
  const target = String(name || '').trim()
  if (!target) return null
  return bankList.find((item) => item.name === target) || null
}

export function getBankLogo(name: string): string {
  return getBankMeta(name)?.logo || ''
}

export function getBankWordmark(name: string): string {
  return getBankMeta(name)?.wordmark || ''
}
