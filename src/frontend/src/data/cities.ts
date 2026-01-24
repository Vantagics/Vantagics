// World cities data organized by country
// Each country has a list of major cities with coordinates

export interface City {
  name: string;
  nameEn: string;
  lat: number;
  lng: number;
}

export interface Country {
  name: string;
  nameEn: string;
  code: string;
  cities: City[];
}

export const countries: Country[] = [
  // ==================== 亚洲 Asia ====================
  {
    name: "中国",
    nameEn: "China",
    code: "CN",
    cities: [
      // 直辖市
      { name: "北京", nameEn: "Beijing", lat: 39.9042, lng: 116.4074 },
      { name: "上海", nameEn: "Shanghai", lat: 31.2304, lng: 121.4737 },
      { name: "天津", nameEn: "Tianjin", lat: 39.3434, lng: 117.3616 },
      { name: "重庆", nameEn: "Chongqing", lat: 29.4316, lng: 106.9123 },
      // 省会城市
      { name: "广州", nameEn: "Guangzhou", lat: 23.1291, lng: 113.2644 },
      { name: "深圳", nameEn: "Shenzhen", lat: 22.5431, lng: 114.0579 },
      { name: "成都", nameEn: "Chengdu", lat: 30.5728, lng: 104.0668 },
      { name: "杭州", nameEn: "Hangzhou", lat: 30.2741, lng: 120.1551 },
      { name: "武汉", nameEn: "Wuhan", lat: 30.5928, lng: 114.3055 },
      { name: "西安", nameEn: "Xi'an", lat: 34.3416, lng: 108.9398 },
      { name: "南京", nameEn: "Nanjing", lat: 32.0603, lng: 118.7969 },
      { name: "郑州", nameEn: "Zhengzhou", lat: 34.7466, lng: 113.6253 },
      { name: "长沙", nameEn: "Changsha", lat: 28.2282, lng: 112.9388 },
      { name: "沈阳", nameEn: "Shenyang", lat: 41.8057, lng: 123.4315 },
      { name: "昆明", nameEn: "Kunming", lat: 24.8801, lng: 102.8329 },
      { name: "哈尔滨", nameEn: "Harbin", lat: 45.8038, lng: 126.5350 },
      { name: "济南", nameEn: "Jinan", lat: 36.6512, lng: 117.1201 },
      { name: "福州", nameEn: "Fuzhou", lat: 26.0745, lng: 119.2965 },
      { name: "南昌", nameEn: "Nanchang", lat: 28.6820, lng: 115.8579 },
      { name: "合肥", nameEn: "Hefei", lat: 31.8206, lng: 117.2272 },
      { name: "石家庄", nameEn: "Shijiazhuang", lat: 38.0428, lng: 114.5149 },
      { name: "太原", nameEn: "Taiyuan", lat: 37.8706, lng: 112.5489 },
      { name: "长春", nameEn: "Changchun", lat: 43.8171, lng: 125.3235 },
      { name: "呼和浩特", nameEn: "Hohhot", lat: 40.8414, lng: 111.7519 },
      { name: "南宁", nameEn: "Nanning", lat: 22.8170, lng: 108.3665 },
      { name: "贵阳", nameEn: "Guiyang", lat: 26.6470, lng: 106.6302 },
      { name: "兰州", nameEn: "Lanzhou", lat: 36.0611, lng: 103.8343 },
      { name: "海口", nameEn: "Haikou", lat: 20.0440, lng: 110.1999 },
      { name: "银川", nameEn: "Yinchuan", lat: 38.4872, lng: 106.2309 },
      { name: "西宁", nameEn: "Xining", lat: 36.6171, lng: 101.7782 },
      { name: "拉萨", nameEn: "Lhasa", lat: 29.6500, lng: 91.1000 },
      { name: "乌鲁木齐", nameEn: "Urumqi", lat: 43.8256, lng: 87.6168 },
      // 主要城市
      { name: "苏州", nameEn: "Suzhou", lat: 31.2990, lng: 120.5853 },
      { name: "青岛", nameEn: "Qingdao", lat: 36.0671, lng: 120.3826 },
      { name: "大连", nameEn: "Dalian", lat: 38.9140, lng: 121.6147 },
      { name: "厦门", nameEn: "Xiamen", lat: 24.4798, lng: 118.0894 },
      { name: "宁波", nameEn: "Ningbo", lat: 29.8683, lng: 121.5440 },
      { name: "无锡", nameEn: "Wuxi", lat: 31.4912, lng: 120.3119 },
      { name: "佛山", nameEn: "Foshan", lat: 23.0218, lng: 113.1219 },
      { name: "东莞", nameEn: "Dongguan", lat: 23.0430, lng: 113.7633 },
      { name: "珠海", nameEn: "Zhuhai", lat: 22.2710, lng: 113.5767 },
      { name: "温州", nameEn: "Wenzhou", lat: 27.9939, lng: 120.6994 },
      { name: "常州", nameEn: "Changzhou", lat: 31.8106, lng: 119.9741 },
      { name: "烟台", nameEn: "Yantai", lat: 37.4638, lng: 121.4479 },
      { name: "威海", nameEn: "Weihai", lat: 37.5091, lng: 122.1164 },
      { name: "泉州", nameEn: "Quanzhou", lat: 24.8741, lng: 118.6757 },
      { name: "南通", nameEn: "Nantong", lat: 31.9829, lng: 120.8943 },
      { name: "徐州", nameEn: "Xuzhou", lat: 34.2044, lng: 117.2859 },
      { name: "扬州", nameEn: "Yangzhou", lat: 32.3936, lng: 119.4126 },
      { name: "绍兴", nameEn: "Shaoxing", lat: 30.0000, lng: 120.5833 },
      { name: "嘉兴", nameEn: "Jiaxing", lat: 30.7522, lng: 120.7550 },
      { name: "金华", nameEn: "Jinhua", lat: 29.0787, lng: 119.6495 },
      { name: "台州", nameEn: "Taizhou", lat: 28.6561, lng: 121.4205 },
      { name: "惠州", nameEn: "Huizhou", lat: 23.1115, lng: 114.4152 },
      { name: "中山", nameEn: "Zhongshan", lat: 22.5176, lng: 113.3926 },
      { name: "汕头", nameEn: "Shantou", lat: 23.3535, lng: 116.6819 },
      { name: "湛江", nameEn: "Zhanjiang", lat: 21.2707, lng: 110.3594 },
      { name: "洛阳", nameEn: "Luoyang", lat: 34.6197, lng: 112.4540 },
      { name: "唐山", nameEn: "Tangshan", lat: 39.6292, lng: 118.1742 },
      { name: "保定", nameEn: "Baoding", lat: 38.8739, lng: 115.4646 },
      { name: "廊坊", nameEn: "Langfang", lat: 39.5186, lng: 116.6831 },
      { name: "秦皇岛", nameEn: "Qinhuangdao", lat: 39.9354, lng: 119.5996 },
      { name: "邯郸", nameEn: "Handan", lat: 36.6116, lng: 114.5391 },
      { name: "包头", nameEn: "Baotou", lat: 40.6571, lng: 109.8400 },
      { name: "鄂尔多斯", nameEn: "Ordos", lat: 39.6086, lng: 109.7810 },
      { name: "吉林", nameEn: "Jilin", lat: 43.8378, lng: 126.5494 },
      { name: "大庆", nameEn: "Daqing", lat: 46.5898, lng: 125.1036 },
      { name: "桂林", nameEn: "Guilin", lat: 25.2736, lng: 110.2907 },
      { name: "三亚", nameEn: "Sanya", lat: 18.2528, lng: 109.5119 },
      { name: "丽江", nameEn: "Lijiang", lat: 26.8721, lng: 100.2299 },
      { name: "大理", nameEn: "Dali", lat: 25.6065, lng: 100.2679 },
      { name: "西双版纳", nameEn: "Xishuangbanna", lat: 22.0017, lng: 100.7975 },
    ]
  },
  {
    name: "香港",
    nameEn: "Hong Kong",
    code: "HK",
    cities: [
      { name: "香港", nameEn: "Hong Kong", lat: 22.3193, lng: 114.1694 },
    ]
  },
  {
    name: "澳门",
    nameEn: "Macau",
    code: "MO",
    cities: [
      { name: "澳门", nameEn: "Macau", lat: 22.1987, lng: 113.5439 },
    ]
  },
  {
    name: "台湾",
    nameEn: "Taiwan",
    code: "TW",
    cities: [
      { name: "台北", nameEn: "Taipei", lat: 25.0330, lng: 121.5654 },
      { name: "高雄", nameEn: "Kaohsiung", lat: 22.6273, lng: 120.3014 },
      { name: "台中", nameEn: "Taichung", lat: 24.1477, lng: 120.6736 },
      { name: "台南", nameEn: "Tainan", lat: 22.9998, lng: 120.2269 },
      { name: "新北", nameEn: "New Taipei", lat: 25.0169, lng: 121.4628 },
      { name: "桃园", nameEn: "Taoyuan", lat: 24.9936, lng: 121.3010 },
      { name: "新竹", nameEn: "Hsinchu", lat: 24.8138, lng: 120.9675 },
      { name: "基隆", nameEn: "Keelung", lat: 25.1276, lng: 121.7392 },
      { name: "花莲", nameEn: "Hualien", lat: 23.9910, lng: 121.6111 },
    ]
  },
  {
    name: "日本",
    nameEn: "Japan",
    code: "JP",
    cities: [
      { name: "东京", nameEn: "Tokyo", lat: 35.6762, lng: 139.6503 },
      { name: "大阪", nameEn: "Osaka", lat: 34.6937, lng: 135.5023 },
      { name: "京都", nameEn: "Kyoto", lat: 35.0116, lng: 135.7681 },
      { name: "横滨", nameEn: "Yokohama", lat: 35.4437, lng: 139.6380 },
      { name: "名古屋", nameEn: "Nagoya", lat: 35.1815, lng: 136.9066 },
      { name: "神户", nameEn: "Kobe", lat: 34.6901, lng: 135.1956 },
      { name: "福冈", nameEn: "Fukuoka", lat: 33.5904, lng: 130.4017 },
      { name: "札幌", nameEn: "Sapporo", lat: 43.0618, lng: 141.3545 },
      { name: "仙台", nameEn: "Sendai", lat: 38.2682, lng: 140.8694 },
      { name: "广岛", nameEn: "Hiroshima", lat: 34.3853, lng: 132.4553 },
      { name: "北九州", nameEn: "Kitakyushu", lat: 33.8835, lng: 130.8752 },
      { name: "千叶", nameEn: "Chiba", lat: 35.6073, lng: 140.1063 },
      { name: "埼玉", nameEn: "Saitama", lat: 35.8617, lng: 139.6455 },
      { name: "川崎", nameEn: "Kawasaki", lat: 35.5309, lng: 139.7030 },
      { name: "那霸", nameEn: "Naha", lat: 26.2124, lng: 127.6809 },
      { name: "长崎", nameEn: "Nagasaki", lat: 32.7503, lng: 129.8779 },
      { name: "金泽", nameEn: "Kanazawa", lat: 36.5944, lng: 136.6256 },
      { name: "新潟", nameEn: "Niigata", lat: 37.9161, lng: 139.0364 },
      { name: "静冈", nameEn: "Shizuoka", lat: 34.9756, lng: 138.3828 },
      { name: "奈良", nameEn: "Nara", lat: 34.6851, lng: 135.8048 },
    ]
  },
  {
    name: "韩国",
    nameEn: "South Korea",
    code: "KR",
    cities: [
      { name: "首尔", nameEn: "Seoul", lat: 37.5665, lng: 126.9780 },
      { name: "釜山", nameEn: "Busan", lat: 35.1796, lng: 129.0756 },
      { name: "仁川", nameEn: "Incheon", lat: 37.4563, lng: 126.7052 },
      { name: "大邱", nameEn: "Daegu", lat: 35.8714, lng: 128.6014 },
      { name: "大田", nameEn: "Daejeon", lat: 36.3504, lng: 127.3845 },
      { name: "光州", nameEn: "Gwangju", lat: 35.1595, lng: 126.8526 },
      { name: "蔚山", nameEn: "Ulsan", lat: 35.5384, lng: 129.3114 },
      { name: "水原", nameEn: "Suwon", lat: 37.2636, lng: 127.0286 },
      { name: "济州", nameEn: "Jeju", lat: 33.4996, lng: 126.5312 },
      { name: "全州", nameEn: "Jeonju", lat: 35.8242, lng: 127.1480 },
      { name: "清州", nameEn: "Cheongju", lat: 36.6424, lng: 127.4890 },
      { name: "昌原", nameEn: "Changwon", lat: 35.2281, lng: 128.6811 },
    ]
  },
  {
    name: "新加坡",
    nameEn: "Singapore",
    code: "SG",
    cities: [
      { name: "新加坡", nameEn: "Singapore", lat: 1.3521, lng: 103.8198 },
    ]
  },
  {
    name: "泰国",
    nameEn: "Thailand",
    code: "TH",
    cities: [
      { name: "曼谷", nameEn: "Bangkok", lat: 13.7563, lng: 100.5018 },
      { name: "清迈", nameEn: "Chiang Mai", lat: 18.7883, lng: 98.9853 },
      { name: "普吉", nameEn: "Phuket", lat: 7.8804, lng: 98.3923 },
      { name: "芭提雅", nameEn: "Pattaya", lat: 12.9236, lng: 100.8825 },
      { name: "清莱", nameEn: "Chiang Rai", lat: 19.9105, lng: 99.8406 },
      { name: "苏梅岛", nameEn: "Koh Samui", lat: 9.5120, lng: 100.0136 },
      { name: "甲米", nameEn: "Krabi", lat: 8.0863, lng: 98.9063 },
      { name: "华欣", nameEn: "Hua Hin", lat: 12.5684, lng: 99.9577 },
    ]
  },
  {
    name: "马来西亚",
    nameEn: "Malaysia",
    code: "MY",
    cities: [
      { name: "吉隆坡", nameEn: "Kuala Lumpur", lat: 3.1390, lng: 101.6869 },
      { name: "槟城", nameEn: "Penang", lat: 5.4164, lng: 100.3327 },
      { name: "新山", nameEn: "Johor Bahru", lat: 1.4927, lng: 103.7414 },
      { name: "马六甲", nameEn: "Malacca", lat: 2.1896, lng: 102.2501 },
      { name: "怡保", nameEn: "Ipoh", lat: 4.5975, lng: 101.0901 },
      { name: "亚庇", nameEn: "Kota Kinabalu", lat: 5.9804, lng: 116.0735 },
      { name: "古晋", nameEn: "Kuching", lat: 1.5535, lng: 110.3593 },
      { name: "兰卡威", nameEn: "Langkawi", lat: 6.3500, lng: 99.8000 },
    ]
  },
  {
    name: "印度尼西亚",
    nameEn: "Indonesia",
    code: "ID",
    cities: [
      { name: "雅加达", nameEn: "Jakarta", lat: -6.2088, lng: 106.8456 },
      { name: "巴厘岛", nameEn: "Bali", lat: -8.3405, lng: 115.0920 },
      { name: "泗水", nameEn: "Surabaya", lat: -7.2575, lng: 112.7521 },
      { name: "万隆", nameEn: "Bandung", lat: -6.9175, lng: 107.6191 },
      { name: "日惹", nameEn: "Yogyakarta", lat: -7.7956, lng: 110.3695 },
      { name: "棉兰", nameEn: "Medan", lat: 3.5952, lng: 98.6722 },
      { name: "三宝垄", nameEn: "Semarang", lat: -6.9666, lng: 110.4196 },
    ]
  },
  {
    name: "越南",
    nameEn: "Vietnam",
    code: "VN",
    cities: [
      { name: "河内", nameEn: "Hanoi", lat: 21.0285, lng: 105.8542 },
      { name: "胡志明市", nameEn: "Ho Chi Minh City", lat: 10.8231, lng: 106.6297 },
      { name: "岘港", nameEn: "Da Nang", lat: 16.0544, lng: 108.2022 },
      { name: "芽庄", nameEn: "Nha Trang", lat: 12.2388, lng: 109.1967 },
      { name: "会安", nameEn: "Hoi An", lat: 15.8801, lng: 108.3380 },
      { name: "顺化", nameEn: "Hue", lat: 16.4637, lng: 107.5909 },
      { name: "海防", nameEn: "Hai Phong", lat: 20.8449, lng: 106.6881 },
      { name: "下龙湾", nameEn: "Ha Long", lat: 20.9517, lng: 107.0480 },
      { name: "大叻", nameEn: "Da Lat", lat: 11.9404, lng: 108.4583 },
      { name: "富国岛", nameEn: "Phu Quoc", lat: 10.2899, lng: 103.9840 },
    ]
  },
  {
    name: "菲律宾",
    nameEn: "Philippines",
    code: "PH",
    cities: [
      { name: "马尼拉", nameEn: "Manila", lat: 14.5995, lng: 120.9842 },
      { name: "宿务", nameEn: "Cebu", lat: 10.3157, lng: 123.8854 },
      { name: "达沃", nameEn: "Davao", lat: 7.1907, lng: 125.4553 },
      { name: "长滩岛", nameEn: "Boracay", lat: 11.9674, lng: 121.9248 },
      { name: "巴拉望", nameEn: "Palawan", lat: 9.8349, lng: 118.7384 },
      { name: "碧瑶", nameEn: "Baguio", lat: 16.4023, lng: 120.5960 },
    ]
  },
  {
    name: "印度",
    nameEn: "India",
    code: "IN",
    cities: [
      { name: "新德里", nameEn: "New Delhi", lat: 28.6139, lng: 77.2090 },
      { name: "孟买", nameEn: "Mumbai", lat: 19.0760, lng: 72.8777 },
      { name: "班加罗尔", nameEn: "Bangalore", lat: 12.9716, lng: 77.5946 },
      { name: "加尔各答", nameEn: "Kolkata", lat: 22.5726, lng: 88.3639 },
      { name: "金奈", nameEn: "Chennai", lat: 13.0827, lng: 80.2707 },
      { name: "海得拉巴", nameEn: "Hyderabad", lat: 17.3850, lng: 78.4867 },
      { name: "艾哈迈达巴德", nameEn: "Ahmedabad", lat: 23.0225, lng: 72.5714 },
      { name: "浦那", nameEn: "Pune", lat: 18.5204, lng: 73.8567 },
      { name: "斋浦尔", nameEn: "Jaipur", lat: 26.9124, lng: 75.7873 },
      { name: "果阿", nameEn: "Goa", lat: 15.2993, lng: 74.1240 },
      { name: "阿格拉", nameEn: "Agra", lat: 27.1767, lng: 78.0081 },
      { name: "瓦拉纳西", nameEn: "Varanasi", lat: 25.3176, lng: 82.9739 },
    ]
  },
  {
    name: "阿联酋",
    nameEn: "United Arab Emirates",
    code: "AE",
    cities: [
      { name: "迪拜", nameEn: "Dubai", lat: 25.2048, lng: 55.2708 },
      { name: "阿布扎比", nameEn: "Abu Dhabi", lat: 24.4539, lng: 54.3773 },
      { name: "沙迦", nameEn: "Sharjah", lat: 25.3463, lng: 55.4209 },
      { name: "阿治曼", nameEn: "Ajman", lat: 25.4052, lng: 55.5136 },
    ]
  },
  {
    name: "沙特阿拉伯",
    nameEn: "Saudi Arabia",
    code: "SA",
    cities: [
      { name: "利雅得", nameEn: "Riyadh", lat: 24.7136, lng: 46.6753 },
      { name: "吉达", nameEn: "Jeddah", lat: 21.4858, lng: 39.1925 },
      { name: "麦加", nameEn: "Mecca", lat: 21.3891, lng: 39.8579 },
      { name: "麦地那", nameEn: "Medina", lat: 24.5247, lng: 39.5692 },
      { name: "达曼", nameEn: "Dammam", lat: 26.4207, lng: 50.0888 },
    ]
  },
  {
    name: "土耳其",
    nameEn: "Turkey",
    code: "TR",
    cities: [
      { name: "伊斯坦布尔", nameEn: "Istanbul", lat: 41.0082, lng: 28.9784 },
      { name: "安卡拉", nameEn: "Ankara", lat: 39.9334, lng: 32.8597 },
      { name: "伊兹密尔", nameEn: "Izmir", lat: 38.4237, lng: 27.1428 },
      { name: "安塔利亚", nameEn: "Antalya", lat: 36.8969, lng: 30.7133 },
      { name: "布尔萨", nameEn: "Bursa", lat: 40.1885, lng: 29.0610 },
      { name: "卡帕多奇亚", nameEn: "Cappadocia", lat: 38.6431, lng: 34.8289 },
      { name: "费特希耶", nameEn: "Fethiye", lat: 36.6515, lng: 29.1225 },
    ]
  },
  {
    name: "以色列",
    nameEn: "Israel",
    code: "IL",
    cities: [
      { name: "特拉维夫", nameEn: "Tel Aviv", lat: 32.0853, lng: 34.7818 },
      { name: "耶路撒冷", nameEn: "Jerusalem", lat: 31.7683, lng: 35.2137 },
      { name: "海法", nameEn: "Haifa", lat: 32.7940, lng: 34.9896 },
      { name: "埃拉特", nameEn: "Eilat", lat: 29.5577, lng: 34.9519 },
    ]
  },
  // ==================== 欧洲 Europe ====================
  {
    name: "英国",
    nameEn: "United Kingdom",
    code: "GB",
    cities: [
      { name: "伦敦", nameEn: "London", lat: 51.5074, lng: -0.1278 },
      { name: "曼彻斯特", nameEn: "Manchester", lat: 53.4808, lng: -2.2426 },
      { name: "伯明翰", nameEn: "Birmingham", lat: 52.4862, lng: -1.8904 },
      { name: "爱丁堡", nameEn: "Edinburgh", lat: 55.9533, lng: -3.1883 },
      { name: "利物浦", nameEn: "Liverpool", lat: 53.4084, lng: -2.9916 },
      { name: "格拉斯哥", nameEn: "Glasgow", lat: 55.8642, lng: -4.2518 },
      { name: "布里斯托", nameEn: "Bristol", lat: 51.4545, lng: -2.5879 },
      { name: "利兹", nameEn: "Leeds", lat: 53.8008, lng: -1.5491 },
      { name: "牛津", nameEn: "Oxford", lat: 51.7520, lng: -1.2577 },
      { name: "剑桥", nameEn: "Cambridge", lat: 52.2053, lng: 0.1218 },
      { name: "约克", nameEn: "York", lat: 53.9591, lng: -1.0815 },
      { name: "巴斯", nameEn: "Bath", lat: 51.3811, lng: -2.3590 },
    ]
  },
  {
    name: "德国",
    nameEn: "Germany",
    code: "DE",
    cities: [
      { name: "柏林", nameEn: "Berlin", lat: 52.5200, lng: 13.4050 },
      { name: "慕尼黑", nameEn: "Munich", lat: 48.1351, lng: 11.5820 },
      { name: "法兰克福", nameEn: "Frankfurt", lat: 50.1109, lng: 8.6821 },
      { name: "汉堡", nameEn: "Hamburg", lat: 53.5511, lng: 9.9937 },
      { name: "科隆", nameEn: "Cologne", lat: 50.9375, lng: 6.9603 },
      { name: "杜塞尔多夫", nameEn: "Düsseldorf", lat: 51.2277, lng: 6.7735 },
      { name: "斯图加特", nameEn: "Stuttgart", lat: 48.7758, lng: 9.1829 },
      { name: "德累斯顿", nameEn: "Dresden", lat: 51.0504, lng: 13.7373 },
      { name: "莱比锡", nameEn: "Leipzig", lat: 51.3397, lng: 12.3731 },
      { name: "纽伦堡", nameEn: "Nuremberg", lat: 49.4521, lng: 11.0767 },
      { name: "海德堡", nameEn: "Heidelberg", lat: 49.3988, lng: 8.6724 },
    ]
  },
  {
    name: "法国",
    nameEn: "France",
    code: "FR",
    cities: [
      { name: "巴黎", nameEn: "Paris", lat: 48.8566, lng: 2.3522 },
      { name: "马赛", nameEn: "Marseille", lat: 43.2965, lng: 5.3698 },
      { name: "里昂", nameEn: "Lyon", lat: 45.7640, lng: 4.8357 },
      { name: "尼斯", nameEn: "Nice", lat: 43.7102, lng: 7.2620 },
      { name: "波尔多", nameEn: "Bordeaux", lat: 44.8378, lng: -0.5792 },
      { name: "图卢兹", nameEn: "Toulouse", lat: 43.6047, lng: 1.4442 },
      { name: "斯特拉斯堡", nameEn: "Strasbourg", lat: 48.5734, lng: 7.7521 },
      { name: "南特", nameEn: "Nantes", lat: 47.2184, lng: -1.5536 },
      { name: "蒙彼利埃", nameEn: "Montpellier", lat: 43.6108, lng: 3.8767 },
      { name: "戛纳", nameEn: "Cannes", lat: 43.5528, lng: 7.0174 },
      { name: "摩纳哥", nameEn: "Monaco", lat: 43.7384, lng: 7.4246 },
    ]
  },
  {
    name: "意大利",
    nameEn: "Italy",
    code: "IT",
    cities: [
      { name: "罗马", nameEn: "Rome", lat: 41.9028, lng: 12.4964 },
      { name: "米兰", nameEn: "Milan", lat: 45.4642, lng: 9.1900 },
      { name: "威尼斯", nameEn: "Venice", lat: 45.4408, lng: 12.3155 },
      { name: "佛罗伦萨", nameEn: "Florence", lat: 43.7696, lng: 11.2558 },
      { name: "那不勒斯", nameEn: "Naples", lat: 40.8518, lng: 14.2681 },
      { name: "都灵", nameEn: "Turin", lat: 45.0703, lng: 7.6869 },
      { name: "博洛尼亚", nameEn: "Bologna", lat: 44.4949, lng: 11.3426 },
      { name: "热那亚", nameEn: "Genoa", lat: 44.4056, lng: 8.9463 },
      { name: "比萨", nameEn: "Pisa", lat: 43.7228, lng: 10.4017 },
      { name: "维罗纳", nameEn: "Verona", lat: 45.4384, lng: 10.9916 },
      { name: "西西里岛", nameEn: "Sicily", lat: 37.5994, lng: 14.0154 },
    ]
  },
  {
    name: "西班牙",
    nameEn: "Spain",
    code: "ES",
    cities: [
      { name: "马德里", nameEn: "Madrid", lat: 40.4168, lng: -3.7038 },
      { name: "巴塞罗那", nameEn: "Barcelona", lat: 41.3851, lng: 2.1734 },
      { name: "瓦伦西亚", nameEn: "Valencia", lat: 39.4699, lng: -0.3763 },
      { name: "塞维利亚", nameEn: "Seville", lat: 37.3891, lng: -5.9845 },
      { name: "毕尔巴鄂", nameEn: "Bilbao", lat: 43.2630, lng: -2.9350 },
      { name: "马拉加", nameEn: "Málaga", lat: 36.7213, lng: -4.4214 },
      { name: "格拉纳达", nameEn: "Granada", lat: 37.1773, lng: -3.5986 },
      { name: "萨拉戈萨", nameEn: "Zaragoza", lat: 41.6488, lng: -0.8891 },
      { name: "马略卡", nameEn: "Mallorca", lat: 39.6953, lng: 3.0176 },
      { name: "伊维萨", nameEn: "Ibiza", lat: 38.9067, lng: 1.4206 },
    ]
  },
  {
    name: "荷兰",
    nameEn: "Netherlands",
    code: "NL",
    cities: [
      { name: "阿姆斯特丹", nameEn: "Amsterdam", lat: 52.3676, lng: 4.9041 },
      { name: "鹿特丹", nameEn: "Rotterdam", lat: 51.9244, lng: 4.4777 },
      { name: "海牙", nameEn: "The Hague", lat: 52.0705, lng: 4.3007 },
      { name: "乌得勒支", nameEn: "Utrecht", lat: 52.0907, lng: 5.1214 },
      { name: "埃因霍温", nameEn: "Eindhoven", lat: 51.4416, lng: 5.4697 },
    ]
  },
  {
    name: "比利时",
    nameEn: "Belgium",
    code: "BE",
    cities: [
      { name: "布鲁塞尔", nameEn: "Brussels", lat: 50.8503, lng: 4.3517 },
      { name: "安特卫普", nameEn: "Antwerp", lat: 51.2194, lng: 4.4025 },
      { name: "布鲁日", nameEn: "Bruges", lat: 51.2093, lng: 3.2247 },
      { name: "根特", nameEn: "Ghent", lat: 51.0543, lng: 3.7174 },
    ]
  },
  {
    name: "瑞士",
    nameEn: "Switzerland",
    code: "CH",
    cities: [
      { name: "苏黎世", nameEn: "Zurich", lat: 47.3769, lng: 8.5417 },
      { name: "日内瓦", nameEn: "Geneva", lat: 46.2044, lng: 6.1432 },
      { name: "伯尔尼", nameEn: "Bern", lat: 46.9480, lng: 7.4474 },
      { name: "巴塞尔", nameEn: "Basel", lat: 47.5596, lng: 7.5886 },
      { name: "洛桑", nameEn: "Lausanne", lat: 46.5197, lng: 6.6323 },
      { name: "卢塞恩", nameEn: "Lucerne", lat: 47.0502, lng: 8.3093 },
      { name: "因特拉肯", nameEn: "Interlaken", lat: 46.6863, lng: 7.8632 },
      { name: "采尔马特", nameEn: "Zermatt", lat: 46.0207, lng: 7.7491 },
    ]
  },
  {
    name: "奥地利",
    nameEn: "Austria",
    code: "AT",
    cities: [
      { name: "维也纳", nameEn: "Vienna", lat: 48.2082, lng: 16.3738 },
      { name: "萨尔茨堡", nameEn: "Salzburg", lat: 47.8095, lng: 13.0550 },
      { name: "因斯布鲁克", nameEn: "Innsbruck", lat: 47.2692, lng: 11.4041 },
      { name: "格拉茨", nameEn: "Graz", lat: 47.0707, lng: 15.4395 },
      { name: "林茨", nameEn: "Linz", lat: 48.3069, lng: 14.2858 },
    ]
  },
  {
    name: "瑞典",
    nameEn: "Sweden",
    code: "SE",
    cities: [
      { name: "斯德哥尔摩", nameEn: "Stockholm", lat: 59.3293, lng: 18.0686 },
      { name: "哥德堡", nameEn: "Gothenburg", lat: 57.7089, lng: 11.9746 },
      { name: "马尔默", nameEn: "Malmö", lat: 55.6050, lng: 13.0038 },
      { name: "乌普萨拉", nameEn: "Uppsala", lat: 59.8586, lng: 17.6389 },
    ]
  },
  {
    name: "挪威",
    nameEn: "Norway",
    code: "NO",
    cities: [
      { name: "奥斯陆", nameEn: "Oslo", lat: 59.9139, lng: 10.7522 },
      { name: "卑尔根", nameEn: "Bergen", lat: 60.3913, lng: 5.3221 },
      { name: "特罗姆瑟", nameEn: "Tromsø", lat: 69.6492, lng: 18.9553 },
      { name: "斯塔万格", nameEn: "Stavanger", lat: 58.9700, lng: 5.7331 },
      { name: "特隆赫姆", nameEn: "Trondheim", lat: 63.4305, lng: 10.3951 },
    ]
  },
  {
    name: "丹麦",
    nameEn: "Denmark",
    code: "DK",
    cities: [
      { name: "哥本哈根", nameEn: "Copenhagen", lat: 55.6761, lng: 12.5683 },
      { name: "奥胡斯", nameEn: "Aarhus", lat: 56.1629, lng: 10.2039 },
      { name: "欧登塞", nameEn: "Odense", lat: 55.4038, lng: 10.4024 },
    ]
  },
  {
    name: "芬兰",
    nameEn: "Finland",
    code: "FI",
    cities: [
      { name: "赫尔辛基", nameEn: "Helsinki", lat: 60.1699, lng: 24.9384 },
      { name: "坦佩雷", nameEn: "Tampere", lat: 61.4978, lng: 23.7610 },
      { name: "图尔库", nameEn: "Turku", lat: 60.4518, lng: 22.2666 },
      { name: "罗瓦涅米", nameEn: "Rovaniemi", lat: 66.5039, lng: 25.7294 },
    ]
  },
  {
    name: "冰岛",
    nameEn: "Iceland",
    code: "IS",
    cities: [
      { name: "雷克雅未克", nameEn: "Reykjavik", lat: 64.1466, lng: -21.9426 },
      { name: "阿克雷里", nameEn: "Akureyri", lat: 65.6885, lng: -18.1262 },
    ]
  },
  {
    name: "俄罗斯",
    nameEn: "Russia",
    code: "RU",
    cities: [
      { name: "莫斯科", nameEn: "Moscow", lat: 55.7558, lng: 37.6173 },
      { name: "圣彼得堡", nameEn: "Saint Petersburg", lat: 59.9311, lng: 30.3609 },
      { name: "新西伯利亚", nameEn: "Novosibirsk", lat: 55.0084, lng: 82.9357 },
      { name: "叶卡捷琳堡", nameEn: "Yekaterinburg", lat: 56.8389, lng: 60.6057 },
      { name: "喀山", nameEn: "Kazan", lat: 55.8304, lng: 49.0661 },
      { name: "索契", nameEn: "Sochi", lat: 43.6028, lng: 39.7342 },
      { name: "海参崴", nameEn: "Vladivostok", lat: 43.1332, lng: 131.9113 },
      { name: "伊尔库茨克", nameEn: "Irkutsk", lat: 52.2870, lng: 104.3050 },
    ]
  },
  {
    name: "波兰",
    nameEn: "Poland",
    code: "PL",
    cities: [
      { name: "华沙", nameEn: "Warsaw", lat: 52.2297, lng: 21.0122 },
      { name: "克拉科夫", nameEn: "Krakow", lat: 50.0647, lng: 19.9450 },
      { name: "格但斯克", nameEn: "Gdańsk", lat: 54.3520, lng: 18.6466 },
      { name: "弗罗茨瓦夫", nameEn: "Wrocław", lat: 51.1079, lng: 17.0385 },
      { name: "波兹南", nameEn: "Poznań", lat: 52.4064, lng: 16.9252 },
    ]
  },
  {
    name: "捷克",
    nameEn: "Czech Republic",
    code: "CZ",
    cities: [
      { name: "布拉格", nameEn: "Prague", lat: 50.0755, lng: 14.4378 },
      { name: "布尔诺", nameEn: "Brno", lat: 49.1951, lng: 16.6068 },
      { name: "卡罗维发利", nameEn: "Karlovy Vary", lat: 50.2325, lng: 12.8713 },
      { name: "切斯基克鲁姆洛夫", nameEn: "Český Krumlov", lat: 48.8127, lng: 14.3175 },
    ]
  },
  {
    name: "匈牙利",
    nameEn: "Hungary",
    code: "HU",
    cities: [
      { name: "布达佩斯", nameEn: "Budapest", lat: 47.4979, lng: 19.0402 },
      { name: "德布勒森", nameEn: "Debrecen", lat: 47.5316, lng: 21.6273 },
      { name: "塞格德", nameEn: "Szeged", lat: 46.2530, lng: 20.1414 },
    ]
  },
  {
    name: "葡萄牙",
    nameEn: "Portugal",
    code: "PT",
    cities: [
      { name: "里斯本", nameEn: "Lisbon", lat: 38.7223, lng: -9.1393 },
      { name: "波尔图", nameEn: "Porto", lat: 41.1579, lng: -8.6291 },
      { name: "法鲁", nameEn: "Faro", lat: 37.0194, lng: -7.9322 },
      { name: "辛特拉", nameEn: "Sintra", lat: 38.8029, lng: -9.3817 },
      { name: "马德拉", nameEn: "Madeira", lat: 32.6669, lng: -16.9241 },
    ]
  },
  {
    name: "希腊",
    nameEn: "Greece",
    code: "GR",
    cities: [
      { name: "雅典", nameEn: "Athens", lat: 37.9838, lng: 23.7275 },
      { name: "塞萨洛尼基", nameEn: "Thessaloniki", lat: 40.6401, lng: 22.9444 },
      { name: "圣托里尼", nameEn: "Santorini", lat: 36.3932, lng: 25.4615 },
      { name: "米科诺斯", nameEn: "Mykonos", lat: 37.4467, lng: 25.3289 },
      { name: "克里特岛", nameEn: "Crete", lat: 35.2401, lng: 24.8093 },
      { name: "罗德岛", nameEn: "Rhodes", lat: 36.4349, lng: 28.2176 },
    ]
  },
  {
    name: "克罗地亚",
    nameEn: "Croatia",
    code: "HR",
    cities: [
      { name: "萨格勒布", nameEn: "Zagreb", lat: 45.8150, lng: 15.9819 },
      { name: "杜布罗夫尼克", nameEn: "Dubrovnik", lat: 42.6507, lng: 18.0944 },
      { name: "斯普利特", nameEn: "Split", lat: 43.5081, lng: 16.4402 },
      { name: "普拉", nameEn: "Pula", lat: 44.8666, lng: 13.8496 },
    ]
  },
  {
    name: "爱尔兰",
    nameEn: "Ireland",
    code: "IE",
    cities: [
      { name: "都柏林", nameEn: "Dublin", lat: 53.3498, lng: -6.2603 },
      { name: "科克", nameEn: "Cork", lat: 51.8985, lng: -8.4756 },
      { name: "戈尔韦", nameEn: "Galway", lat: 53.2707, lng: -9.0568 },
      { name: "利默里克", nameEn: "Limerick", lat: 52.6638, lng: -8.6267 },
    ]
  },
  // ==================== 北美洲 North America ====================
  {
    name: "美国",
    nameEn: "United States",
    code: "US",
    cities: [
      // 主要城市
      { name: "纽约", nameEn: "New York", lat: 40.7128, lng: -74.0060 },
      { name: "洛杉矶", nameEn: "Los Angeles", lat: 34.0522, lng: -118.2437 },
      { name: "芝加哥", nameEn: "Chicago", lat: 41.8781, lng: -87.6298 },
      { name: "休斯顿", nameEn: "Houston", lat: 29.7604, lng: -95.3698 },
      { name: "旧金山", nameEn: "San Francisco", lat: 37.7749, lng: -122.4194 },
      { name: "西雅图", nameEn: "Seattle", lat: 47.6062, lng: -122.3321 },
      { name: "波士顿", nameEn: "Boston", lat: 42.3601, lng: -71.0589 },
      { name: "华盛顿", nameEn: "Washington D.C.", lat: 38.9072, lng: -77.0369 },
      { name: "迈阿密", nameEn: "Miami", lat: 25.7617, lng: -80.1918 },
      { name: "拉斯维加斯", nameEn: "Las Vegas", lat: 36.1699, lng: -115.1398 },
      { name: "费城", nameEn: "Philadelphia", lat: 39.9526, lng: -75.1652 },
      { name: "达拉斯", nameEn: "Dallas", lat: 32.7767, lng: -96.7970 },
      { name: "亚特兰大", nameEn: "Atlanta", lat: 33.7490, lng: -84.3880 },
      { name: "丹佛", nameEn: "Denver", lat: 39.7392, lng: -104.9903 },
      { name: "凤凰城", nameEn: "Phoenix", lat: 33.4484, lng: -112.0740 },
      { name: "圣地亚哥", nameEn: "San Diego", lat: 32.7157, lng: -117.1611 },
      { name: "奥斯汀", nameEn: "Austin", lat: 30.2672, lng: -97.7431 },
      { name: "波特兰", nameEn: "Portland", lat: 45.5152, lng: -122.6784 },
      { name: "底特律", nameEn: "Detroit", lat: 42.3314, lng: -83.0458 },
      { name: "明尼阿波利斯", nameEn: "Minneapolis", lat: 44.9778, lng: -93.2650 },
      { name: "新奥尔良", nameEn: "New Orleans", lat: 29.9511, lng: -90.0715 },
      { name: "盐湖城", nameEn: "Salt Lake City", lat: 40.7608, lng: -111.8910 },
      { name: "圣何塞", nameEn: "San Jose", lat: 37.3382, lng: -121.8863 },
      { name: "奥兰多", nameEn: "Orlando", lat: 28.5383, lng: -81.3792 },
      { name: "夏威夷檀香山", nameEn: "Honolulu", lat: 21.3069, lng: -157.8583 },
      { name: "安克雷奇", nameEn: "Anchorage", lat: 61.2181, lng: -149.9003 },
      { name: "纳什维尔", nameEn: "Nashville", lat: 36.1627, lng: -86.7816 },
      { name: "印第安纳波利斯", nameEn: "Indianapolis", lat: 39.7684, lng: -86.1581 },
      { name: "夏洛特", nameEn: "Charlotte", lat: 35.2271, lng: -80.8431 },
      { name: "圣安东尼奥", nameEn: "San Antonio", lat: 29.4241, lng: -98.4936 },
    ]
  },
  {
    name: "加拿大",
    nameEn: "Canada",
    code: "CA",
    cities: [
      { name: "多伦多", nameEn: "Toronto", lat: 43.6532, lng: -79.3832 },
      { name: "温哥华", nameEn: "Vancouver", lat: 49.2827, lng: -123.1207 },
      { name: "蒙特利尔", nameEn: "Montreal", lat: 45.5017, lng: -73.5673 },
      { name: "卡尔加里", nameEn: "Calgary", lat: 51.0447, lng: -114.0719 },
      { name: "渥太华", nameEn: "Ottawa", lat: 45.4215, lng: -75.6972 },
      { name: "埃德蒙顿", nameEn: "Edmonton", lat: 53.5461, lng: -113.4938 },
      { name: "魁北克城", nameEn: "Quebec City", lat: 46.8139, lng: -71.2080 },
      { name: "温尼伯", nameEn: "Winnipeg", lat: 49.8951, lng: -97.1384 },
      { name: "哈利法克斯", nameEn: "Halifax", lat: 44.6488, lng: -63.5752 },
      { name: "维多利亚", nameEn: "Victoria", lat: 48.4284, lng: -123.3656 },
      { name: "惠斯勒", nameEn: "Whistler", lat: 50.1163, lng: -122.9574 },
      { name: "班夫", nameEn: "Banff", lat: 51.1784, lng: -115.5708 },
    ]
  },
  {
    name: "墨西哥",
    nameEn: "Mexico",
    code: "MX",
    cities: [
      { name: "墨西哥城", nameEn: "Mexico City", lat: 19.4326, lng: -99.1332 },
      { name: "坎昆", nameEn: "Cancún", lat: 21.1619, lng: -86.8515 },
      { name: "瓜达拉哈拉", nameEn: "Guadalajara", lat: 20.6597, lng: -103.3496 },
      { name: "蒙特雷", nameEn: "Monterrey", lat: 25.6866, lng: -100.3161 },
      { name: "普拉亚德尔卡门", nameEn: "Playa del Carmen", lat: 20.6296, lng: -87.0739 },
      { name: "蒂华纳", nameEn: "Tijuana", lat: 32.5149, lng: -117.0382 },
      { name: "洛斯卡沃斯", nameEn: "Los Cabos", lat: 22.8905, lng: -109.9167 },
      { name: "瓦哈卡", nameEn: "Oaxaca", lat: 17.0732, lng: -96.7266 },
      { name: "普埃布拉", nameEn: "Puebla", lat: 19.0414, lng: -98.2063 },
    ]
  },
  // ==================== 南美洲 South America ====================
  {
    name: "巴西",
    nameEn: "Brazil",
    code: "BR",
    cities: [
      { name: "圣保罗", nameEn: "São Paulo", lat: -23.5505, lng: -46.6333 },
      { name: "里约热内卢", nameEn: "Rio de Janeiro", lat: -22.9068, lng: -43.1729 },
      { name: "巴西利亚", nameEn: "Brasília", lat: -15.8267, lng: -47.9218 },
      { name: "萨尔瓦多", nameEn: "Salvador", lat: -12.9714, lng: -38.5014 },
      { name: "福塔莱萨", nameEn: "Fortaleza", lat: -3.7172, lng: -38.5433 },
      { name: "贝洛奥里藏特", nameEn: "Belo Horizonte", lat: -19.9167, lng: -43.9345 },
      { name: "马瑙斯", nameEn: "Manaus", lat: -3.1190, lng: -60.0217 },
      { name: "库里蒂巴", nameEn: "Curitiba", lat: -25.4290, lng: -49.2671 },
      { name: "累西腓", nameEn: "Recife", lat: -8.0476, lng: -34.8770 },
      { name: "弗洛里亚诺波利斯", nameEn: "Florianópolis", lat: -27.5954, lng: -48.5480 },
    ]
  },
  {
    name: "阿根廷",
    nameEn: "Argentina",
    code: "AR",
    cities: [
      { name: "布宜诺斯艾利斯", nameEn: "Buenos Aires", lat: -34.6037, lng: -58.3816 },
      { name: "科尔多瓦", nameEn: "Córdoba", lat: -31.4201, lng: -64.1888 },
      { name: "门多萨", nameEn: "Mendoza", lat: -32.8895, lng: -68.8458 },
      { name: "巴里洛切", nameEn: "Bariloche", lat: -41.1335, lng: -71.3103 },
      { name: "乌斯怀亚", nameEn: "Ushuaia", lat: -54.8019, lng: -68.3030 },
      { name: "伊瓜苏", nameEn: "Iguazu", lat: -25.5972, lng: -54.5786 },
    ]
  },
  {
    name: "智利",
    nameEn: "Chile",
    code: "CL",
    cities: [
      { name: "圣地亚哥", nameEn: "Santiago", lat: -33.4489, lng: -70.6693 },
      { name: "瓦尔帕莱索", nameEn: "Valparaíso", lat: -33.0472, lng: -71.6127 },
      { name: "复活节岛", nameEn: "Easter Island", lat: -27.1127, lng: -109.3497 },
      { name: "蓬塔阿雷纳斯", nameEn: "Punta Arenas", lat: -53.1638, lng: -70.9171 },
    ]
  },
  {
    name: "秘鲁",
    nameEn: "Peru",
    code: "PE",
    cities: [
      { name: "利马", nameEn: "Lima", lat: -12.0464, lng: -77.0428 },
      { name: "库斯科", nameEn: "Cusco", lat: -13.5319, lng: -71.9675 },
      { name: "马丘比丘", nameEn: "Machu Picchu", lat: -13.1631, lng: -72.5450 },
      { name: "阿雷基帕", nameEn: "Arequipa", lat: -16.4090, lng: -71.5375 },
    ]
  },
  {
    name: "哥伦比亚",
    nameEn: "Colombia",
    code: "CO",
    cities: [
      { name: "波哥大", nameEn: "Bogotá", lat: 4.7110, lng: -74.0721 },
      { name: "麦德林", nameEn: "Medellín", lat: 6.2442, lng: -75.5812 },
      { name: "卡塔赫纳", nameEn: "Cartagena", lat: 10.3910, lng: -75.4794 },
      { name: "卡利", nameEn: "Cali", lat: 3.4516, lng: -76.5320 },
    ]
  },
  // ==================== 大洋洲 Oceania ====================
  {
    name: "澳大利亚",
    nameEn: "Australia",
    code: "AU",
    cities: [
      { name: "悉尼", nameEn: "Sydney", lat: -33.8688, lng: 151.2093 },
      { name: "墨尔本", nameEn: "Melbourne", lat: -37.8136, lng: 144.9631 },
      { name: "布里斯班", nameEn: "Brisbane", lat: -27.4698, lng: 153.0251 },
      { name: "珀斯", nameEn: "Perth", lat: -31.9505, lng: 115.8605 },
      { name: "阿德莱德", nameEn: "Adelaide", lat: -34.9285, lng: 138.6007 },
      { name: "堪培拉", nameEn: "Canberra", lat: -35.2809, lng: 149.1300 },
      { name: "黄金海岸", nameEn: "Gold Coast", lat: -28.0167, lng: 153.4000 },
      { name: "凯恩斯", nameEn: "Cairns", lat: -16.9186, lng: 145.7781 },
      { name: "达尔文", nameEn: "Darwin", lat: -12.4634, lng: 130.8456 },
      { name: "霍巴特", nameEn: "Hobart", lat: -42.8821, lng: 147.3272 },
      { name: "大堡礁", nameEn: "Great Barrier Reef", lat: -18.2871, lng: 147.6992 },
      { name: "乌鲁鲁", nameEn: "Uluru", lat: -25.3444, lng: 131.0369 },
    ]
  },
  {
    name: "新西兰",
    nameEn: "New Zealand",
    code: "NZ",
    cities: [
      { name: "奥克兰", nameEn: "Auckland", lat: -36.8509, lng: 174.7645 },
      { name: "惠灵顿", nameEn: "Wellington", lat: -41.2865, lng: 174.7762 },
      { name: "基督城", nameEn: "Christchurch", lat: -43.5321, lng: 172.6362 },
      { name: "皇后镇", nameEn: "Queenstown", lat: -45.0312, lng: 168.6626 },
      { name: "罗托鲁瓦", nameEn: "Rotorua", lat: -38.1368, lng: 176.2497 },
      { name: "但尼丁", nameEn: "Dunedin", lat: -45.8788, lng: 170.5028 },
    ]
  },
  {
    name: "斐济",
    nameEn: "Fiji",
    code: "FJ",
    cities: [
      { name: "苏瓦", nameEn: "Suva", lat: -18.1416, lng: 178.4419 },
      { name: "楠迪", nameEn: "Nadi", lat: -17.7765, lng: 177.4356 },
    ]
  },
  // ==================== 非洲 Africa ====================
  {
    name: "南非",
    nameEn: "South Africa",
    code: "ZA",
    cities: [
      { name: "约翰内斯堡", nameEn: "Johannesburg", lat: -26.2041, lng: 28.0473 },
      { name: "开普敦", nameEn: "Cape Town", lat: -33.9249, lng: 18.4241 },
      { name: "德班", nameEn: "Durban", lat: -29.8587, lng: 31.0218 },
      { name: "比勒陀利亚", nameEn: "Pretoria", lat: -25.7479, lng: 28.2293 },
      { name: "伊丽莎白港", nameEn: "Port Elizabeth", lat: -33.9608, lng: 25.6022 },
    ]
  },
  {
    name: "埃及",
    nameEn: "Egypt",
    code: "EG",
    cities: [
      { name: "开罗", nameEn: "Cairo", lat: 30.0444, lng: 31.2357 },
      { name: "亚历山大", nameEn: "Alexandria", lat: 31.2001, lng: 29.9187 },
      { name: "卢克索", nameEn: "Luxor", lat: 25.6872, lng: 32.6396 },
      { name: "阿斯旺", nameEn: "Aswan", lat: 24.0889, lng: 32.8998 },
      { name: "沙姆沙伊赫", nameEn: "Sharm El Sheikh", lat: 27.9158, lng: 34.3300 },
      { name: "赫尔格达", nameEn: "Hurghada", lat: 27.2579, lng: 33.8116 },
    ]
  },
  {
    name: "摩洛哥",
    nameEn: "Morocco",
    code: "MA",
    cities: [
      { name: "卡萨布兰卡", nameEn: "Casablanca", lat: 33.5731, lng: -7.5898 },
      { name: "马拉喀什", nameEn: "Marrakech", lat: 31.6295, lng: -7.9811 },
      { name: "非斯", nameEn: "Fes", lat: 34.0181, lng: -5.0078 },
      { name: "拉巴特", nameEn: "Rabat", lat: 34.0209, lng: -6.8416 },
      { name: "丹吉尔", nameEn: "Tangier", lat: 35.7595, lng: -5.8340 },
      { name: "舍夫沙万", nameEn: "Chefchaouen", lat: 35.1688, lng: -5.2636 },
    ]
  },
  {
    name: "肯尼亚",
    nameEn: "Kenya",
    code: "KE",
    cities: [
      { name: "内罗毕", nameEn: "Nairobi", lat: -1.2921, lng: 36.8219 },
      { name: "蒙巴萨", nameEn: "Mombasa", lat: -4.0435, lng: 39.6682 },
      { name: "马赛马拉", nameEn: "Masai Mara", lat: -1.4061, lng: 35.0168 },
    ]
  },
  {
    name: "坦桑尼亚",
    nameEn: "Tanzania",
    code: "TZ",
    cities: [
      { name: "达累斯萨拉姆", nameEn: "Dar es Salaam", lat: -6.7924, lng: 39.2083 },
      { name: "桑给巴尔", nameEn: "Zanzibar", lat: -6.1659, lng: 39.2026 },
      { name: "阿鲁沙", nameEn: "Arusha", lat: -3.3869, lng: 36.6830 },
    ]
  },
  {
    name: "尼日利亚",
    nameEn: "Nigeria",
    code: "NG",
    cities: [
      { name: "拉各斯", nameEn: "Lagos", lat: 6.5244, lng: 3.3792 },
      { name: "阿布贾", nameEn: "Abuja", lat: 9.0765, lng: 7.3986 },
    ]
  },
  {
    name: "埃塞俄比亚",
    nameEn: "Ethiopia",
    code: "ET",
    cities: [
      { name: "亚的斯亚贝巴", nameEn: "Addis Ababa", lat: 9.0320, lng: 38.7469 },
    ]
  },
  {
    name: "毛里求斯",
    nameEn: "Mauritius",
    code: "MU",
    cities: [
      { name: "路易港", nameEn: "Port Louis", lat: -20.1609, lng: 57.5012 },
    ]
  },
  {
    name: "塞舌尔",
    nameEn: "Seychelles",
    code: "SC",
    cities: [
      { name: "维多利亚", nameEn: "Victoria", lat: -4.6191, lng: 55.4513 },
    ]
  },
  {
    name: "马尔代夫",
    nameEn: "Maldives",
    code: "MV",
    cities: [
      { name: "马累", nameEn: "Malé", lat: 4.1755, lng: 73.5093 },
    ]
  },
  // ==================== 更多亚洲国家 More Asian Countries ====================
  {
    name: "蒙古",
    nameEn: "Mongolia",
    code: "MN",
    cities: [
      { name: "乌兰巴托", nameEn: "Ulaanbaatar", lat: 47.8864, lng: 106.9057 },
    ]
  },
  {
    name: "朝鲜",
    nameEn: "North Korea",
    code: "KP",
    cities: [
      { name: "平壤", nameEn: "Pyongyang", lat: 39.0392, lng: 125.7625 },
    ]
  },
  {
    name: "缅甸",
    nameEn: "Myanmar",
    code: "MM",
    cities: [
      { name: "仰光", nameEn: "Yangon", lat: 16.8661, lng: 96.1951 },
      { name: "曼德勒", nameEn: "Mandalay", lat: 21.9588, lng: 96.0891 },
      { name: "内比都", nameEn: "Naypyidaw", lat: 19.7633, lng: 96.0785 },
      { name: "蒲甘", nameEn: "Bagan", lat: 21.1717, lng: 94.8585 },
    ]
  },
  {
    name: "柬埔寨",
    nameEn: "Cambodia",
    code: "KH",
    cities: [
      { name: "金边", nameEn: "Phnom Penh", lat: 11.5564, lng: 104.9282 },
      { name: "暹粒", nameEn: "Siem Reap", lat: 13.3671, lng: 103.8448 },
      { name: "西哈努克", nameEn: "Sihanoukville", lat: 10.6093, lng: 103.5296 },
    ]
  },
  {
    name: "老挝",
    nameEn: "Laos",
    code: "LA",
    cities: [
      { name: "万象", nameEn: "Vientiane", lat: 17.9757, lng: 102.6331 },
      { name: "琅勃拉邦", nameEn: "Luang Prabang", lat: 19.8830, lng: 102.1347 },
    ]
  },
  {
    name: "尼泊尔",
    nameEn: "Nepal",
    code: "NP",
    cities: [
      { name: "加德满都", nameEn: "Kathmandu", lat: 27.7172, lng: 85.3240 },
      { name: "博卡拉", nameEn: "Pokhara", lat: 28.2096, lng: 83.9856 },
    ]
  },
  {
    name: "不丹",
    nameEn: "Bhutan",
    code: "BT",
    cities: [
      { name: "廷布", nameEn: "Thimphu", lat: 27.4728, lng: 89.6390 },
      { name: "帕罗", nameEn: "Paro", lat: 27.4305, lng: 89.4125 },
    ]
  },
  {
    name: "斯里兰卡",
    nameEn: "Sri Lanka",
    code: "LK",
    cities: [
      { name: "科伦坡", nameEn: "Colombo", lat: 6.9271, lng: 79.8612 },
      { name: "康提", nameEn: "Kandy", lat: 7.2906, lng: 80.6337 },
      { name: "加勒", nameEn: "Galle", lat: 6.0535, lng: 80.2210 },
      { name: "努瓦勒埃利耶", nameEn: "Nuwara Eliya", lat: 6.9497, lng: 80.7891 },
    ]
  },
  {
    name: "孟加拉国",
    nameEn: "Bangladesh",
    code: "BD",
    cities: [
      { name: "达卡", nameEn: "Dhaka", lat: 23.8103, lng: 90.4125 },
      { name: "吉大港", nameEn: "Chittagong", lat: 22.3569, lng: 91.7832 },
    ]
  },
  {
    name: "巴基斯坦",
    nameEn: "Pakistan",
    code: "PK",
    cities: [
      { name: "伊斯兰堡", nameEn: "Islamabad", lat: 33.6844, lng: 73.0479 },
      { name: "卡拉奇", nameEn: "Karachi", lat: 24.8607, lng: 67.0011 },
      { name: "拉合尔", nameEn: "Lahore", lat: 31.5204, lng: 74.3587 },
      { name: "白沙瓦", nameEn: "Peshawar", lat: 34.0151, lng: 71.5249 },
    ]
  },
  {
    name: "伊朗",
    nameEn: "Iran",
    code: "IR",
    cities: [
      { name: "德黑兰", nameEn: "Tehran", lat: 35.6892, lng: 51.3890 },
      { name: "伊斯法罕", nameEn: "Isfahan", lat: 32.6546, lng: 51.6680 },
      { name: "设拉子", nameEn: "Shiraz", lat: 29.5918, lng: 52.5837 },
      { name: "马什哈德", nameEn: "Mashhad", lat: 36.2605, lng: 59.6168 },
    ]
  },
  {
    name: "伊拉克",
    nameEn: "Iraq",
    code: "IQ",
    cities: [
      { name: "巴格达", nameEn: "Baghdad", lat: 33.3152, lng: 44.3661 },
      { name: "巴士拉", nameEn: "Basra", lat: 30.5085, lng: 47.7804 },
      { name: "埃尔比勒", nameEn: "Erbil", lat: 36.1901, lng: 44.0091 },
    ]
  },
  {
    name: "约旦",
    nameEn: "Jordan",
    code: "JO",
    cities: [
      { name: "安曼", nameEn: "Amman", lat: 31.9454, lng: 35.9284 },
      { name: "佩特拉", nameEn: "Petra", lat: 30.3285, lng: 35.4444 },
      { name: "亚喀巴", nameEn: "Aqaba", lat: 29.5267, lng: 35.0078 },
    ]
  },
  {
    name: "黎巴嫩",
    nameEn: "Lebanon",
    code: "LB",
    cities: [
      { name: "贝鲁特", nameEn: "Beirut", lat: 33.8938, lng: 35.5018 },
    ]
  },
  {
    name: "科威特",
    nameEn: "Kuwait",
    code: "KW",
    cities: [
      { name: "科威特城", nameEn: "Kuwait City", lat: 29.3759, lng: 47.9774 },
    ]
  },
  {
    name: "卡塔尔",
    nameEn: "Qatar",
    code: "QA",
    cities: [
      { name: "多哈", nameEn: "Doha", lat: 25.2854, lng: 51.5310 },
    ]
  },
  {
    name: "巴林",
    nameEn: "Bahrain",
    code: "BH",
    cities: [
      { name: "麦纳麦", nameEn: "Manama", lat: 26.2285, lng: 50.5860 },
    ]
  },
  {
    name: "阿曼",
    nameEn: "Oman",
    code: "OM",
    cities: [
      { name: "马斯喀特", nameEn: "Muscat", lat: 23.5880, lng: 58.3829 },
      { name: "塞拉莱", nameEn: "Salalah", lat: 17.0151, lng: 54.0924 },
    ]
  },
  {
    name: "也门",
    nameEn: "Yemen",
    code: "YE",
    cities: [
      { name: "萨那", nameEn: "Sanaa", lat: 15.3694, lng: 44.1910 },
      { name: "亚丁", nameEn: "Aden", lat: 12.7855, lng: 45.0187 },
    ]
  },
  {
    name: "哈萨克斯坦",
    nameEn: "Kazakhstan",
    code: "KZ",
    cities: [
      { name: "阿斯塔纳", nameEn: "Astana", lat: 51.1694, lng: 71.4491 },
      { name: "阿拉木图", nameEn: "Almaty", lat: 43.2220, lng: 76.8512 },
    ]
  },
  {
    name: "乌兹别克斯坦",
    nameEn: "Uzbekistan",
    code: "UZ",
    cities: [
      { name: "塔什干", nameEn: "Tashkent", lat: 41.2995, lng: 69.2401 },
      { name: "撒马尔罕", nameEn: "Samarkand", lat: 39.6542, lng: 66.9597 },
      { name: "布哈拉", nameEn: "Bukhara", lat: 39.7681, lng: 64.4556 },
    ]
  },
  {
    name: "土库曼斯坦",
    nameEn: "Turkmenistan",
    code: "TM",
    cities: [
      { name: "阿什哈巴德", nameEn: "Ashgabat", lat: 37.9601, lng: 58.3261 },
    ]
  },
  {
    name: "吉尔吉斯斯坦",
    nameEn: "Kyrgyzstan",
    code: "KG",
    cities: [
      { name: "比什凯克", nameEn: "Bishkek", lat: 42.8746, lng: 74.5698 },
    ]
  },
  {
    name: "塔吉克斯坦",
    nameEn: "Tajikistan",
    code: "TJ",
    cities: [
      { name: "杜尚别", nameEn: "Dushanbe", lat: 38.5598, lng: 68.7740 },
    ]
  },
  {
    name: "阿塞拜疆",
    nameEn: "Azerbaijan",
    code: "AZ",
    cities: [
      { name: "巴库", nameEn: "Baku", lat: 40.4093, lng: 49.8671 },
    ]
  },
  {
    name: "格鲁吉亚",
    nameEn: "Georgia",
    code: "GE",
    cities: [
      { name: "第比利斯", nameEn: "Tbilisi", lat: 41.7151, lng: 44.8271 },
      { name: "巴统", nameEn: "Batumi", lat: 41.6168, lng: 41.6367 },
    ]
  },
  {
    name: "亚美尼亚",
    nameEn: "Armenia",
    code: "AM",
    cities: [
      { name: "埃里温", nameEn: "Yerevan", lat: 40.1792, lng: 44.4991 },
    ]
  },
  {
    name: "塞浦路斯",
    nameEn: "Cyprus",
    code: "CY",
    cities: [
      { name: "尼科西亚", nameEn: "Nicosia", lat: 35.1856, lng: 33.3823 },
      { name: "利马索尔", nameEn: "Limassol", lat: 34.7071, lng: 33.0226 },
      { name: "帕福斯", nameEn: "Paphos", lat: 34.7754, lng: 32.4245 },
    ]
  },
  // ==================== 更多欧洲国家 More European Countries ====================
  {
    name: "乌克兰",
    nameEn: "Ukraine",
    code: "UA",
    cities: [
      { name: "基辅", nameEn: "Kyiv", lat: 50.4501, lng: 30.5234 },
      { name: "利沃夫", nameEn: "Lviv", lat: 49.8397, lng: 24.0297 },
      { name: "敖德萨", nameEn: "Odesa", lat: 46.4825, lng: 30.7233 },
      { name: "哈尔科夫", nameEn: "Kharkiv", lat: 49.9935, lng: 36.2304 },
    ]
  },
  {
    name: "白俄罗斯",
    nameEn: "Belarus",
    code: "BY",
    cities: [
      { name: "明斯克", nameEn: "Minsk", lat: 53.9006, lng: 27.5590 },
    ]
  },
  {
    name: "摩尔多瓦",
    nameEn: "Moldova",
    code: "MD",
    cities: [
      { name: "基希讷乌", nameEn: "Chișinău", lat: 47.0105, lng: 28.8638 },
    ]
  },
  {
    name: "罗马尼亚",
    nameEn: "Romania",
    code: "RO",
    cities: [
      { name: "布加勒斯特", nameEn: "Bucharest", lat: 44.4268, lng: 26.1025 },
      { name: "克卢日-纳波卡", nameEn: "Cluj-Napoca", lat: 46.7712, lng: 23.6236 },
      { name: "布拉索夫", nameEn: "Brașov", lat: 45.6427, lng: 25.5887 },
      { name: "锡比乌", nameEn: "Sibiu", lat: 45.7983, lng: 24.1256 },
    ]
  },
  {
    name: "保加利亚",
    nameEn: "Bulgaria",
    code: "BG",
    cities: [
      { name: "索非亚", nameEn: "Sofia", lat: 42.6977, lng: 23.3219 },
      { name: "普罗夫迪夫", nameEn: "Plovdiv", lat: 42.1354, lng: 24.7453 },
      { name: "瓦尔纳", nameEn: "Varna", lat: 43.2141, lng: 27.9147 },
    ]
  },
  {
    name: "塞尔维亚",
    nameEn: "Serbia",
    code: "RS",
    cities: [
      { name: "贝尔格莱德", nameEn: "Belgrade", lat: 44.7866, lng: 20.4489 },
      { name: "诺维萨德", nameEn: "Novi Sad", lat: 45.2671, lng: 19.8335 },
    ]
  },
  {
    name: "斯洛文尼亚",
    nameEn: "Slovenia",
    code: "SI",
    cities: [
      { name: "卢布尔雅那", nameEn: "Ljubljana", lat: 46.0569, lng: 14.5058 },
      { name: "布莱德", nameEn: "Bled", lat: 46.3683, lng: 14.1146 },
    ]
  },
  {
    name: "斯洛伐克",
    nameEn: "Slovakia",
    code: "SK",
    cities: [
      { name: "布拉迪斯拉发", nameEn: "Bratislava", lat: 48.1486, lng: 17.1077 },
      { name: "科希策", nameEn: "Košice", lat: 48.7164, lng: 21.2611 },
    ]
  },
  {
    name: "波黑",
    nameEn: "Bosnia and Herzegovina",
    code: "BA",
    cities: [
      { name: "萨拉热窝", nameEn: "Sarajevo", lat: 43.8563, lng: 18.4131 },
      { name: "莫斯塔尔", nameEn: "Mostar", lat: 43.3438, lng: 17.8078 },
    ]
  },
  {
    name: "黑山",
    nameEn: "Montenegro",
    code: "ME",
    cities: [
      { name: "波德戈里察", nameEn: "Podgorica", lat: 42.4304, lng: 19.2594 },
      { name: "科托尔", nameEn: "Kotor", lat: 42.4247, lng: 18.7712 },
      { name: "布德瓦", nameEn: "Budva", lat: 42.2864, lng: 18.8400 },
    ]
  },
  {
    name: "北马其顿",
    nameEn: "North Macedonia",
    code: "MK",
    cities: [
      { name: "斯科普里", nameEn: "Skopje", lat: 41.9981, lng: 21.4254 },
      { name: "奥赫里德", nameEn: "Ohrid", lat: 41.1231, lng: 20.8016 },
    ]
  },
  {
    name: "阿尔巴尼亚",
    nameEn: "Albania",
    code: "AL",
    cities: [
      { name: "地拉那", nameEn: "Tirana", lat: 41.3275, lng: 19.8187 },
      { name: "萨兰达", nameEn: "Saranda", lat: 39.8661, lng: 20.0050 },
    ]
  },
  {
    name: "科索沃",
    nameEn: "Kosovo",
    code: "XK",
    cities: [
      { name: "普里什蒂纳", nameEn: "Pristina", lat: 42.6629, lng: 21.1655 },
    ]
  },
  {
    name: "爱沙尼亚",
    nameEn: "Estonia",
    code: "EE",
    cities: [
      { name: "塔林", nameEn: "Tallinn", lat: 59.4370, lng: 24.7536 },
      { name: "塔尔图", nameEn: "Tartu", lat: 58.3780, lng: 26.7290 },
    ]
  },
  {
    name: "拉脱维亚",
    nameEn: "Latvia",
    code: "LV",
    cities: [
      { name: "里加", nameEn: "Riga", lat: 56.9496, lng: 24.1052 },
    ]
  },
  {
    name: "立陶宛",
    nameEn: "Lithuania",
    code: "LT",
    cities: [
      { name: "维尔纽斯", nameEn: "Vilnius", lat: 54.6872, lng: 25.2797 },
      { name: "考纳斯", nameEn: "Kaunas", lat: 54.8985, lng: 23.9036 },
    ]
  },
  {
    name: "卢森堡",
    nameEn: "Luxembourg",
    code: "LU",
    cities: [
      { name: "卢森堡市", nameEn: "Luxembourg City", lat: 49.6116, lng: 6.1319 },
    ]
  },
  {
    name: "马耳他",
    nameEn: "Malta",
    code: "MT",
    cities: [
      { name: "瓦莱塔", nameEn: "Valletta", lat: 35.8989, lng: 14.5146 },
    ]
  },
  {
    name: "安道尔",
    nameEn: "Andorra",
    code: "AD",
    cities: [
      { name: "安道尔城", nameEn: "Andorra la Vella", lat: 42.5063, lng: 1.5218 },
    ]
  },
  {
    name: "列支敦士登",
    nameEn: "Liechtenstein",
    code: "LI",
    cities: [
      { name: "瓦杜兹", nameEn: "Vaduz", lat: 47.1410, lng: 9.5209 },
    ]
  },
  {
    name: "圣马力诺",
    nameEn: "San Marino",
    code: "SM",
    cities: [
      { name: "圣马力诺", nameEn: "San Marino", lat: 43.9424, lng: 12.4578 },
    ]
  },
  {
    name: "梵蒂冈",
    nameEn: "Vatican City",
    code: "VA",
    cities: [
      { name: "梵蒂冈城", nameEn: "Vatican City", lat: 41.9029, lng: 12.4534 },
    ]
  },
  // ==================== 更多非洲国家 More African Countries ====================
  {
    name: "阿尔及利亚",
    nameEn: "Algeria",
    code: "DZ",
    cities: [
      { name: "阿尔及尔", nameEn: "Algiers", lat: 36.7538, lng: 3.0588 },
      { name: "奥兰", nameEn: "Oran", lat: 35.6969, lng: -0.6331 },
    ]
  },
  {
    name: "突尼斯",
    nameEn: "Tunisia",
    code: "TN",
    cities: [
      { name: "突尼斯市", nameEn: "Tunis", lat: 36.8065, lng: 10.1815 },
      { name: "苏塞", nameEn: "Sousse", lat: 35.8288, lng: 10.6405 },
      { name: "杰尔巴岛", nameEn: "Djerba", lat: 33.8076, lng: 10.8451 },
    ]
  },
  {
    name: "利比亚",
    nameEn: "Libya",
    code: "LY",
    cities: [
      { name: "的黎波里", nameEn: "Tripoli", lat: 32.8872, lng: 13.1913 },
      { name: "班加西", nameEn: "Benghazi", lat: 32.1194, lng: 20.0868 },
    ]
  },
  {
    name: "苏丹",
    nameEn: "Sudan",
    code: "SD",
    cities: [
      { name: "喀土穆", nameEn: "Khartoum", lat: 15.5007, lng: 32.5599 },
    ]
  },
  {
    name: "加纳",
    nameEn: "Ghana",
    code: "GH",
    cities: [
      { name: "阿克拉", nameEn: "Accra", lat: 5.6037, lng: -0.1870 },
    ]
  },
  {
    name: "科特迪瓦",
    nameEn: "Ivory Coast",
    code: "CI",
    cities: [
      { name: "阿比让", nameEn: "Abidjan", lat: 5.3600, lng: -4.0083 },
    ]
  },
  {
    name: "塞内加尔",
    nameEn: "Senegal",
    code: "SN",
    cities: [
      { name: "达喀尔", nameEn: "Dakar", lat: 14.7167, lng: -17.4677 },
    ]
  },
  {
    name: "喀麦隆",
    nameEn: "Cameroon",
    code: "CM",
    cities: [
      { name: "雅温得", nameEn: "Yaoundé", lat: 3.8480, lng: 11.5021 },
      { name: "杜阿拉", nameEn: "Douala", lat: 4.0511, lng: 9.7679 },
    ]
  },
  {
    name: "刚果民主共和国",
    nameEn: "DR Congo",
    code: "CD",
    cities: [
      { name: "金沙萨", nameEn: "Kinshasa", lat: -4.4419, lng: 15.2663 },
    ]
  },
  {
    name: "安哥拉",
    nameEn: "Angola",
    code: "AO",
    cities: [
      { name: "罗安达", nameEn: "Luanda", lat: -8.8390, lng: 13.2894 },
    ]
  },
  {
    name: "莫桑比克",
    nameEn: "Mozambique",
    code: "MZ",
    cities: [
      { name: "马普托", nameEn: "Maputo", lat: -25.9692, lng: 32.5732 },
    ]
  },
  {
    name: "津巴布韦",
    nameEn: "Zimbabwe",
    code: "ZW",
    cities: [
      { name: "哈拉雷", nameEn: "Harare", lat: -17.8252, lng: 31.0335 },
      { name: "维多利亚瀑布", nameEn: "Victoria Falls", lat: -17.9243, lng: 25.8572 },
    ]
  },
  {
    name: "赞比亚",
    nameEn: "Zambia",
    code: "ZM",
    cities: [
      { name: "卢萨卡", nameEn: "Lusaka", lat: -15.3875, lng: 28.3228 },
    ]
  },
  {
    name: "博茨瓦纳",
    nameEn: "Botswana",
    code: "BW",
    cities: [
      { name: "哈博罗内", nameEn: "Gaborone", lat: -24.6282, lng: 25.9231 },
    ]
  },
  {
    name: "纳米比亚",
    nameEn: "Namibia",
    code: "NA",
    cities: [
      { name: "温得和克", nameEn: "Windhoek", lat: -22.5609, lng: 17.0658 },
    ]
  },
  {
    name: "乌干达",
    nameEn: "Uganda",
    code: "UG",
    cities: [
      { name: "坎帕拉", nameEn: "Kampala", lat: 0.3476, lng: 32.5825 },
    ]
  },
  {
    name: "卢旺达",
    nameEn: "Rwanda",
    code: "RW",
    cities: [
      { name: "基加利", nameEn: "Kigali", lat: -1.9403, lng: 29.8739 },
    ]
  },
  {
    name: "马达加斯加",
    nameEn: "Madagascar",
    code: "MG",
    cities: [
      { name: "塔那那利佛", nameEn: "Antananarivo", lat: -18.8792, lng: 47.5079 },
    ]
  },
  // ==================== 中美洲和加勒比海 Central America & Caribbean ====================
  {
    name: "古巴",
    nameEn: "Cuba",
    code: "CU",
    cities: [
      { name: "哈瓦那", nameEn: "Havana", lat: 23.1136, lng: -82.3666 },
      { name: "巴拉德罗", nameEn: "Varadero", lat: 23.1394, lng: -81.2861 },
      { name: "圣地亚哥", nameEn: "Santiago de Cuba", lat: 20.0247, lng: -75.8219 },
    ]
  },
  {
    name: "牙买加",
    nameEn: "Jamaica",
    code: "JM",
    cities: [
      { name: "金斯敦", nameEn: "Kingston", lat: 18.0179, lng: -76.8099 },
      { name: "蒙特哥贝", nameEn: "Montego Bay", lat: 18.4762, lng: -77.8939 },
    ]
  },
  {
    name: "多米尼加",
    nameEn: "Dominican Republic",
    code: "DO",
    cities: [
      { name: "圣多明各", nameEn: "Santo Domingo", lat: 18.4861, lng: -69.9312 },
      { name: "蓬塔卡纳", nameEn: "Punta Cana", lat: 18.5601, lng: -68.3725 },
    ]
  },
  {
    name: "波多黎各",
    nameEn: "Puerto Rico",
    code: "PR",
    cities: [
      { name: "圣胡安", nameEn: "San Juan", lat: 18.4655, lng: -66.1057 },
    ]
  },
  {
    name: "巴哈马",
    nameEn: "Bahamas",
    code: "BS",
    cities: [
      { name: "拿骚", nameEn: "Nassau", lat: 25.0343, lng: -77.3963 },
    ]
  },
  {
    name: "巴巴多斯",
    nameEn: "Barbados",
    code: "BB",
    cities: [
      { name: "布里奇敦", nameEn: "Bridgetown", lat: 13.1132, lng: -59.5988 },
    ]
  },
  {
    name: "特立尼达和多巴哥",
    nameEn: "Trinidad and Tobago",
    code: "TT",
    cities: [
      { name: "西班牙港", nameEn: "Port of Spain", lat: 10.6596, lng: -61.5086 },
    ]
  },
  {
    name: "阿鲁巴",
    nameEn: "Aruba",
    code: "AW",
    cities: [
      { name: "奥拉涅斯塔德", nameEn: "Oranjestad", lat: 12.5092, lng: -70.0086 },
    ]
  },
  {
    name: "库拉索",
    nameEn: "Curaçao",
    code: "CW",
    cities: [
      { name: "威廉斯塔德", nameEn: "Willemstad", lat: 12.1696, lng: -68.9900 },
    ]
  },
  {
    name: "开曼群岛",
    nameEn: "Cayman Islands",
    code: "KY",
    cities: [
      { name: "乔治敦", nameEn: "George Town", lat: 19.2866, lng: -81.3744 },
    ]
  },
  {
    name: "危地马拉",
    nameEn: "Guatemala",
    code: "GT",
    cities: [
      { name: "危地马拉城", nameEn: "Guatemala City", lat: 14.6349, lng: -90.5069 },
      { name: "安提瓜", nameEn: "Antigua Guatemala", lat: 14.5586, lng: -90.7295 },
    ]
  },
  {
    name: "伯利兹",
    nameEn: "Belize",
    code: "BZ",
    cities: [
      { name: "贝尔莫潘", nameEn: "Belmopan", lat: 17.2510, lng: -88.7590 },
      { name: "伯利兹城", nameEn: "Belize City", lat: 17.4986, lng: -88.1886 },
    ]
  },
  {
    name: "洪都拉斯",
    nameEn: "Honduras",
    code: "HN",
    cities: [
      { name: "特古西加尔巴", nameEn: "Tegucigalpa", lat: 14.0723, lng: -87.1921 },
    ]
  },
  {
    name: "萨尔瓦多",
    nameEn: "El Salvador",
    code: "SV",
    cities: [
      { name: "圣萨尔瓦多", nameEn: "San Salvador", lat: 13.6929, lng: -89.2182 },
    ]
  },
  {
    name: "尼加拉瓜",
    nameEn: "Nicaragua",
    code: "NI",
    cities: [
      { name: "马那瓜", nameEn: "Managua", lat: 12.1149, lng: -86.2362 },
    ]
  },
  {
    name: "哥斯达黎加",
    nameEn: "Costa Rica",
    code: "CR",
    cities: [
      { name: "圣何塞", nameEn: "San José", lat: 9.9281, lng: -84.0907 },
      { name: "利蒙", nameEn: "Limón", lat: 9.9907, lng: -83.0359 },
    ]
  },
  {
    name: "巴拿马",
    nameEn: "Panama",
    code: "PA",
    cities: [
      { name: "巴拿马城", nameEn: "Panama City", lat: 8.9824, lng: -79.5199 },
    ]
  },
  // ==================== 更多南美洲国家 More South American Countries ====================
  {
    name: "委内瑞拉",
    nameEn: "Venezuela",
    code: "VE",
    cities: [
      { name: "加拉加斯", nameEn: "Caracas", lat: 10.4806, lng: -66.9036 },
    ]
  },
  {
    name: "厄瓜多尔",
    nameEn: "Ecuador",
    code: "EC",
    cities: [
      { name: "基多", nameEn: "Quito", lat: -0.1807, lng: -78.4678 },
      { name: "瓜亚基尔", nameEn: "Guayaquil", lat: -2.1894, lng: -79.8891 },
      { name: "加拉帕戈斯", nameEn: "Galápagos", lat: -0.9538, lng: -90.9656 },
    ]
  },
  {
    name: "玻利维亚",
    nameEn: "Bolivia",
    code: "BO",
    cities: [
      { name: "拉巴斯", nameEn: "La Paz", lat: -16.4897, lng: -68.1193 },
      { name: "苏克雷", nameEn: "Sucre", lat: -19.0196, lng: -65.2619 },
      { name: "乌尤尼", nameEn: "Uyuni", lat: -20.4631, lng: -66.8253 },
    ]
  },
  {
    name: "巴拉圭",
    nameEn: "Paraguay",
    code: "PY",
    cities: [
      { name: "亚松森", nameEn: "Asunción", lat: -25.2637, lng: -57.5759 },
    ]
  },
  {
    name: "乌拉圭",
    nameEn: "Uruguay",
    code: "UY",
    cities: [
      { name: "蒙得维的亚", nameEn: "Montevideo", lat: -34.9011, lng: -56.1645 },
      { name: "埃斯特角城", nameEn: "Punta del Este", lat: -34.9667, lng: -54.9500 },
    ]
  },
  {
    name: "圭亚那",
    nameEn: "Guyana",
    code: "GY",
    cities: [
      { name: "乔治敦", nameEn: "Georgetown", lat: 6.8013, lng: -58.1551 },
    ]
  },
  {
    name: "苏里南",
    nameEn: "Suriname",
    code: "SR",
    cities: [
      { name: "帕拉马里博", nameEn: "Paramaribo", lat: 5.8520, lng: -55.2038 },
    ]
  },
  // ==================== 太平洋岛国 Pacific Islands ====================
  {
    name: "帕劳",
    nameEn: "Palau",
    code: "PW",
    cities: [
      { name: "科罗尔", nameEn: "Koror", lat: 7.3419, lng: 134.4792 },
    ]
  },
  {
    name: "关岛",
    nameEn: "Guam",
    code: "GU",
    cities: [
      { name: "阿加尼亚", nameEn: "Hagåtña", lat: 13.4443, lng: 144.7937 },
    ]
  },
  {
    name: "塞班岛",
    nameEn: "Saipan",
    code: "MP",
    cities: [
      { name: "塞班", nameEn: "Saipan", lat: 15.1900, lng: 145.7500 },
    ]
  },
  {
    name: "大溪地",
    nameEn: "Tahiti",
    code: "PF",
    cities: [
      { name: "帕皮提", nameEn: "Papeete", lat: -17.5516, lng: -149.5585 },
      { name: "波拉波拉", nameEn: "Bora Bora", lat: -16.5004, lng: -151.7415 },
    ]
  },
  {
    name: "萨摩亚",
    nameEn: "Samoa",
    code: "WS",
    cities: [
      { name: "阿皮亚", nameEn: "Apia", lat: -13.8333, lng: -171.7500 },
    ]
  },
  {
    name: "汤加",
    nameEn: "Tonga",
    code: "TO",
    cities: [
      { name: "努库阿洛法", nameEn: "Nukuʻalofa", lat: -21.2114, lng: -175.1998 },
    ]
  },
  {
    name: "瓦努阿图",
    nameEn: "Vanuatu",
    code: "VU",
    cities: [
      { name: "维拉港", nameEn: "Port Vila", lat: -17.7333, lng: 168.3167 },
    ]
  },
  {
    name: "新喀里多尼亚",
    nameEn: "New Caledonia",
    code: "NC",
    cities: [
      { name: "努美阿", nameEn: "Nouméa", lat: -22.2758, lng: 166.4580 },
    ]
  },
  {
    name: "巴布亚新几内亚",
    nameEn: "Papua New Guinea",
    code: "PG",
    cities: [
      { name: "莫尔兹比港", nameEn: "Port Moresby", lat: -9.4438, lng: 147.1803 },
    ]
  },
];

// Helper function to get city display name based on language
export const getCityDisplayName = (city: City, language: string): string => {
  return language === 'zh' ? city.name : city.nameEn;
};

// Helper function to get country display name based on language
export const getCountryDisplayName = (country: Country, language: string): string => {
  return language === 'zh' ? country.name : country.nameEn;
};
