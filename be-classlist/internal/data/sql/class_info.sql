CREATE TABLE class_info (
                            id VARCHAR(255) NOT NULL PRIMARY KEY,          -- 课程ID，作为主键
                            created_at TIMESTAMP,                         -- 创建时间
                            updated_at TIMESTAMP,                         -- 更新时间
                            jxb_id VARCHAR(100) NOT NULL,                 -- 教学班ID
                            day INT NOT NULL,                             -- 星期几
                            teacher VARCHAR(255) NOT NULL,                -- 任课教师
                            `where` VARCHAR(255) NOT NULL,                -- 上课地点
                            class_when VARCHAR(255) NOT NULL,             -- 上课是第几节
                            week_duration VARCHAR(255) NOT NULL,          -- 上课的周数
                            class_name VARCHAR(255) NOT NULL,             -- 课程名称
                            credit FLOAT DEFAULT 1.0,                     -- 学分，默认为1
                            weeks INT NOT NULL,                           -- 哪些周
                            semester VARCHAR(1) NOT NULL,                 -- 学期
                            year VARCHAR(5) NOT NULL,                     -- 学年

    -- 索引
                            INDEX idx_time (year, semester,created_at)   -- 创建的联合索引，按学年、学期、时间排序
);
