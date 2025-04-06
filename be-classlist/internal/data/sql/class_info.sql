SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for class_info
-- ----------------------------
CREATE TABLE `class_info`  (
  `id` varchar(150) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,  -- 课程ID
  `created_at` datetime(3) NULL DEFAULT NULL,   -- 创建时间
  `updated_at` datetime(3) NULL DEFAULT NULL,   -- 更新时间
  `jxb_id` varchar(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NULL DEFAULT NULL,--教学班ID
  `day` bigint NOT NULL,--星期
  `teacher` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,--老师姓名
  `where` varchar(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,    --上课地点
  `class_when` varchar(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,--上课时间(1-2,3-4这种)
  `week_duration` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,--课程的周数(文字描述)
  `class_name` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,--课程名称
  `credit` double NULL DEFAULT 1,--课程学分
  `weeks` bigint NOT NULL,--课程周数,用二进制数表示(如001代表第一周,011代表1,2周,111代表1-3周)
  `semester` varchar(1) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,--学期名
  `year` varchar(5) CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci NOT NULL,--学年名
  PRIMARY KEY (`id`) USING BTREE
) ENGINE = InnoDB CHARACTER SET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci ROW_FORMAT = Dynamic;

SET FOREIGN_KEY_CHECKS = 1;
