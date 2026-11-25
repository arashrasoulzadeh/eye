package service
import ("testing";"reflect";"context";"sync";"github.com/arashrasoulzadeh/eye/internal/model")
func TestDimensionValues(t *testing.T){
 if got:=dimensionValues([]any{"a","b","a"},true);!reflect.DeepEqual(got,[]string{"a","b"}){t.Fatal(got)}
 if got:=dimensionValues(nil,true);!reflect.DeepEqual(got,[]string{"(missing)"}){t.Fatal(got)}
}

func TestNestedDimensionPaths(t *testing.T){
 got:=flattenDimensions(map[string]any{"device":map[string]any{"os":"Android"},"a/b":42.0,"a~b":true},"",0)
 if got["/device/os"]!="Android"||got["/a~1b"]!=42.0||got["/a~0b"]!=true{t.Fatal(got)}
}
func TestDimensionFilters(t *testing.T){
 dims:=map[string]any{"tags":[]any{"Math","Science"},"score":0.0}
 for _,c:=range []struct{f DimensionFilter;want bool}{
 {DimensionFilter{Key:"tags",Mode:"contains",Value:"math"},true},
 {DimensionFilter{Key:"tags",Mode:"equal",Value:"Science"},true},
 {DimensionFilter{Key:"tags",Mode:"not_equal",Value:"Science"},false},
 {DimensionFilter{Key:"score",Mode:"exists"},true},
 {DimensionFilter{Key:"unknown",Mode:"missing"},true},
 }{if got:=matchDimensionFilter(dims,c.f);got!=c.want{t.Errorf("%+v: %v",c.f,got)}}
}

func TestDimensionPairStats(t *testing.T) {
 var positive,negative,constant,small dimensionPairStats
 for i:=0;i<100;i++{x:=float64(i)+1e9;positive.add(x,2*x);negative.add(x,-x);constant.add(x,1);if i<10{small.add(x,x)}}
 if r,ok:=positive.coefficient();!ok||r<0.999999{t.Fatalf("positive: %v %v",r,ok)}
 if r,ok:=negative.coefficient();!ok||r> -0.999999{t.Fatalf("negative: %v %v",r,ok)}
 if _,ok:=constant.coefficient();ok{t.Fatal("constant series must not correlate")}
 if _,ok:=small.coefficient();ok{t.Fatal("small sample must not correlate")}
}

func TestDimensionSummaryScope(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 events:=[]model.Event{
 {EventID:"a",TenantID:"t1",UserID:"u1",SessionID:"s1",EventName:"click",ClientTS:100},
 {EventID:"b",TenantID:"t1",UserID:"u1",SessionID:"s2",EventName:"click",ClientTS:101},
 {EventID:"c",TenantID:"t1",SessionID:"s2",EventName:"click",ClientTS:102},
 {EventID:"d",TenantID:"t1",UserID:"u2",EventName:"view",ClientTS:103},
 {EventID:"e",TenantID:"t2",UserID:"u3",SessionID:"s3",EventName:"click",ClientTS:104},
 {EventID:"f",TenantID:"t1",UserID:"u4",SessionID:"s4",EventName:"click",ClientTS:99},
 }
 if err:=db.Create(&events).Error;err!=nil{t.Fatal(err)}
 q:=&QueryService{DB:db}
 got,err:=q.AnalyzeDimensions(context.Background(),DimensionInput{Tenant:"t1",From:100,To:104,Types:[]string{"click"},Operation:"count",Summary:true})
 if err!=nil{t.Fatal(err)}
 if got.Matched!=3||got.Users!=1||got.Sessions!=2||got.IdentifiedEvents!=2{t.Fatalf("unexpected summary: %+v",got)}
}

func TestCombinedDimensionSummaryTrend(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 events:=[]model.Event{
 {EventID:"a",TenantID:"t1",UserID:"u1",SessionID:"s1",EventName:"click",ClientTS:100},
 {EventID:"b",TenantID:"t1",UserID:"u1",SessionID:"s2",EventName:"click",ClientTS:3600000},
 {EventID:"c",TenantID:"t1",EventName:"click",ClientTS:3600001},
 {EventID:"d",TenantID:"t2",EventName:"click",ClientTS:3600001},
 {EventID:"e",TenantID:"t1",EventName:"view",ClientTS:3600001},
 {EventID:"f",TenantID:"t1",EventName:"click",ClientTS:7200000},
 }
 if err:=db.Create(&events).Error;err!=nil{t.Fatal(err)}
 q:=&QueryService{DB:db}
 got,err:=q.AnalyzeDimensions(context.Background(),DimensionInput{Tenant:"t1",From:0,To:7199999,Types:[]string{"click"},Operation:"count",Summary:true,IncludeTrend:true,TimeBucket:"hour"})
 if err!=nil{t.Fatal(err)}
 if got.Matched!=3||got.Users!=1||got.Sessions!=2||got.IdentifiedEvents!=2||got.Execution!="database"{t.Fatalf("unexpected summary: %+v",got)}
 if len(got.Rows)!=2||got.Rows[0].Group!="1970-01-01 00:00"||got.Rows[0].Count!=1||got.Rows[1].Group!="1970-01-01 01:00"||got.Rows[1].Count!=2{t.Fatalf("unexpected buckets: %+v",got.Rows)}
}

func TestDimensionSummaryCacheIsolation(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 events:=[]model.Event{
 {EventID:"cache-a",TenantID:"t1",EventName:"click",ClientTS:100},
 {EventID:"cache-b",TenantID:"t2",EventName:"click",ClientTS:100},
 {EventID:"cache-c",TenantID:"t2",EventName:"view",ClientTS:101},
 }
 if err:=db.Create(&events).Error;err!=nil{t.Fatal(err)}
 q:=&QueryService{DB:db}
 for _,tc:=range []struct{tenant string;types []string;want int}{
 {"t1",nil,1},{"t2",nil,2},{"t2",[]string{"click"},1},{"t2",[]string{"missing"},0},{"t1",nil,1},
 }{
  got,err:=q.AnalyzeDimensions(context.Background(),DimensionInput{Tenant:tc.tenant,From:0,To:200,Types:tc.types,Operation:"count",Summary:true,CacheSummary:true})
  if err!=nil{t.Fatal(err)}
  if got.Matched!=tc.want{t.Fatalf("tenant %s types %v: got %d want %d",tc.tenant,tc.types,got.Matched,tc.want)}
 }
}
func TestDimensionSummaryEmptyPeriod(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 q:=&QueryService{DB:db}
 got,err:=q.AnalyzeDimensions(context.Background(),DimensionInput{Tenant:"empty",From:0,To:3600000,Operation:"count",Summary:true,IncludeTrend:true,TimeBucket:"hour"})
 if err!=nil{t.Fatal(err)}
 if got.Matched!=0||got.Users!=0||got.Sessions!=0||len(got.Rows)!=0{t.Fatalf("unexpected empty result: %+v",got)}
}

func TestDimensionSummaryCacheResultOwnership(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 if err:=db.Create(&model.Event{EventID:"ownership",TenantID:"t1",EventName:"click",ClientTS:100}).Error;err!=nil{t.Fatal(err)}
 q:=&QueryService{DB:db}
 in:=DimensionInput{Tenant:"t1",From:0,To:200,Operation:"count",Summary:true,CacheSummary:true,IncludeTrend:true,TimeBucket:"hour"}
 first,err:=q.AnalyzeDimensions(context.Background(),in)
 if err!=nil{t.Fatal(err)}
 if len(first.Rows)!=1{t.Fatalf("unexpected rows: %+v",first.Rows)}
 first.Rows[0].Count=999
 second,err:=q.AnalyzeDimensions(context.Background(),in)
 if err!=nil{t.Fatal(err)}
 if len(second.Rows)!=1||second.Rows[0].Count!=1{t.Fatalf("cache mutated: %+v",second.Rows)}
 second.Rows[0].Count=888
 third,err:=q.AnalyzeDimensions(context.Background(),in)
 if err!=nil{t.Fatal(err)}
 if len(third.Rows)!=1||third.Rows[0].Count!=1{t.Fatalf("cache hit mutated: %+v",third.Rows)}
}
func TestDimensionSummaryCacheCancelled(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 q:=&QueryService{DB:db}
 in:=DimensionInput{Tenant:"t1",From:0,To:200,Operation:"count",Summary:true,CacheSummary:true}
 if _,err:=q.AnalyzeDimensions(context.Background(),in);err!=nil{t.Fatal(err)}
 ctx,cancel:=context.WithCancel(context.Background());cancel()
 if _,err:=q.AnalyzeDimensions(ctx,in);err!=context.Canceled{t.Fatalf("expected cancellation, got %v",err)}
}

func TestDimensionSummaryConcurrentCacheHits(t *testing.T) {
 db:=newTestDB_dashboard_cache_test(t)
 if err:=db.Create(&model.Event{EventID:"concurrent",TenantID:"t1",EventName:"click",ClientTS:100}).Error;err!=nil{t.Fatal(err)}
 q:=&QueryService{DB:db}
 in:=DimensionInput{Tenant:"t1",From:0,To:200,Operation:"count",Summary:true,CacheSummary:true,IncludeTrend:true,TimeBucket:"hour"}
 if _,err:=q.AnalyzeDimensions(context.Background(),in);err!=nil{t.Fatal(err)}
 var wg sync.WaitGroup
 for i:=0;i<64;i++{wg.Add(1);go func(){defer wg.Done();got,err:=q.AnalyzeDimensions(context.Background(),in);if err!=nil{t.Error(err);return};if got.Matched!=1||len(got.Rows)!=1||got.Rows[0].Count!=1{t.Errorf("unexpected concurrent result: %+v",got);return};got.Rows[0].Count=999}()}
 wg.Wait()
}
